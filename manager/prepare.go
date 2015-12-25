package manager

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	p2p "registry_p2p"
	"registry_p2p/bittorrent"
	"strconv"
	"strings"
	"sync"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

type tagId map[string]string
type repositories map[string]tagId

func Prepare(mg *Manager, task *Task) (err error) {
	log.Printf("++pull: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("++pull: %s\n", task.ImageName))
	if err = Pull(mg.DockerClient, task.ImageName, task.Username, task.Password, task.Email); err != nil {
		return
	}
	log.Printf("--pull: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("--pull: %s\n", task.ImageName))

	id, err := p2p.GetImageIDByImageName(mg.DockerClient, task.ImageName)
	if err != nil {
		return
	}

	task.ImageID = id

	switch task.Mode {
	case p2p.MODE_IMAGE:
		return prepareForImage(mg, task)
	case p2p.MODE_LAYER:
		return prepareForLayer(mg, task)
	case p2p.MODE_MULTI_LAYER:
		return prepareForMultiLayer(mg, task)
	default:
		return errors.New("unsupported task mode")
	}

	return
}

func prepareForImage(mg *Manager, task *Task) (err error) {

	task.URL = mg.FileServerPrefix + "image/" + task.ImageID + ".torrent"

image:
	packageExist, packagePath, err := mg.PackageExist(task.ImageID, "image")
	if err != nil {
		return
	}

	if !packageExist {

		c, err := mg.PoolAdd("image" + "_" + task.ImageID)
		if err != nil {
			if c != nil {
				log.Printf("image is being exported by other progress, please wait: %s", task.ImageName)
				task.Writer.Write(fmt.Sprintf("image is being exported by other progress, please wait: %s \n", task.ImageName))
				<-c
				goto image
			} else {
				return err
			}
		}
		defer mg.PoolDelete("image" + "_" + task.ImageName)

		f, err := os.Create(packagePath)
		if err != nil {
			return err
		}

		gw := gzip.NewWriter(f)

		log.Printf("++save image: %s", task.ImageName)
		task.Writer.Write(fmt.Sprintf("++save image: %s \n", task.ImageName))
		if err = Save(mg.DockerClient, task.ImageName, gw); err != nil {
			return err
		}
		gw.Close()
		f.Close()

		task.Writer.Write(fmt.Sprintf("--save image: %s \n", task.ImageName))
		log.Printf("--save image: %s", task.ImageName)
	} else {
		task.Writer.Write(fmt.Sprintf("skip save image: %s \n", task.ImageName))
		log.Printf("skip save image: %s", task.ImageName)
	}

	torrentExist, torrentPath, err := mg.TorrentExist(task.ImageID, "image")
	if err != nil {
		return
	}

	if !torrentExist {
		log.Printf("++create torrent: %s", task.ImageName)
		task.Writer.Write(fmt.Sprintf("++create torrent: %s \n", task.ImageName))
		if err = CreateTorrent(mg.BTClient, packagePath, torrentPath, mg.Trackers); err != nil {
			return err
		}
		task.Writer.Write(fmt.Sprintf("--create torrent: %s \n", task.ImageName))
		log.Printf("--create torrent: %s", task.ImageName)
	} else {
		task.Writer.Write(fmt.Sprintf("skip create torrent: %s \n", task.ImageName))
		log.Printf("skip create torrent: %s", task.ImageName)
	}

	config := make(map[string]string)
	config["target"] = "manager"

	log.Printf("++load to bt client: %s", task.ImageName)
	task.Writer.Write(fmt.Sprintf("++load to bt client: %s \n", task.ImageName))
	if err = Download(mg.BTClient, packagePath, torrentPath, config); err != nil {
		return
	}
	task.Writer.Write(fmt.Sprintf("--load to bt client: %s \n", task.ImageName))
	log.Printf("--load to bt client: %s", task.ImageName)

	return
}

func prepareForLayer(mg *Manager, task *Task) (err error) {
	id := task.ImageID

	for len(id) != 0 {
		parentID, err := p2p.GetParentID(mg.DockerClient, id)
		if err != nil {
			return err
		}

		task.Items = append(task.Items, &p2p.Item{
			ID:       id,
			ParentID: parentID,
			Type:     "layer",
			URL:      mg.FileServerPrefix + "layer/" + id + ".torrent",
		})

		id = parentID
	}

	allLayerExist := true

	for _, item := range task.Items {
		packageExist, _, err := mg.PackageExist(item.ID, item.Type)
		if err != nil {
			return err
		}
		if !packageExist {
			allLayerExist = false
			break
		}
	}

	if !allLayerExist {

		dir := filepath.Join(os.TempDir(), strconv.Itoa(int(time.Now().Unix())))
		if err = os.Mkdir(dir, 644); err != nil {
			return err
		}
		defer os.RemoveAll(dir)

		path := filepath.Join(dir, task.ImageID+".tar")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		log.Printf("++save image: %s", task.ImageName)
		task.Writer.Write(fmt.Sprintf("++save image: %s \n", task.ImageName))
		if err = Save(mg.DockerClient, task.ImageName, f); err != nil {
			return err
		}
		if err = f.Sync(); err != nil {
			return err
		}
		task.Writer.Write(fmt.Sprintf("--save image: %s \n", task.ImageName))
		log.Printf("--save image: %s", task.ImageName)

		if _, err = f.Seek(0, 0); err != nil {
			return err
		}

		tr := tar.NewReader(f)

		for {
			header, err := tr.Next()
			if err != nil {
				if err == io.EOF {
					break
				}

				return err
			}

			if header.Typeflag != tar.TypeDir {
				continue
			}

			id := header.Name[:len(header.Name)-1]

		layer:
			packageExist, packagePath, err := mg.PackageExist(id, "layer")
			if err != nil {
				return err
			}

			if !packageExist {
				c, err := mg.PoolAdd("layer_" + id)
				if err != nil {
					if c != nil {
						log.Printf("layer is being extract by other progress, please wait: %s", id)
						task.Writer.Write(fmt.Sprintf("layer is being extract by other progress, please wait: %s \n", id))
						<-c
						goto layer
					} else {
						return err
					}
				}
				defer mg.PoolDelete("layer_" + id)

				log.Printf("++extract layer: %s", id)
				task.Writer.Write(fmt.Sprintf("++extract layer: %s \n", id))

				pf, err := os.Create(packagePath)
				if err != nil {
					return err
				}

				gw := gzip.NewWriter(pf)

				tw := tar.NewWriter(gw)

				if err = tw.WriteHeader(header); err != nil {
					return err
				}

				for i := 0; i < 3; i++ {
					header, err := tr.Next()
					if err != nil {
						return err
					}

					if err = tw.WriteHeader(header); err != nil {
						return err
					}

					if _, err = io.Copy(tw, tr); err != nil {
						return err
					}
				}

				tw.Close()
				gw.Close()
				pf.Close()

				task.Writer.Write(fmt.Sprintf("--extract layer: %s \n", id))
				log.Printf("--extract layer: %s", id)
			} else {
				task.Writer.Write(fmt.Sprintf("skip extract layer: %s \n", id))
				log.Printf("skip extract layer: %s", id)
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(task.Items))

	config := make(map[string]string)
	config["target"] = "manager"
	for _, item := range task.Items {
		torrentExist, torrentPath, err := mg.TorrentExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		packageExist, packagePath, err := mg.PackageExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		if !torrentExist {

			if !packageExist {
				return fmt.Errorf("[ERROR]package not exist: %s", packagePath)
			}

			log.Printf("++make torrent: %s", item.Type+"_"+item.ID)
			task.Writer.Write(fmt.Sprintf("++make torrent: %s \n", item.Type+"_"+item.ID))
			if err = CreateTorrent(mg.BTClient, packagePath, torrentPath, mg.Trackers); err != nil {
				return err
			}
			task.Writer.Write(fmt.Sprintf("--make torrent: %s \n", item.Type+"_"+item.ID))
			log.Printf("--make torrent: %s", item.Type+"_"+item.ID)

		} else {
			task.Writer.Write(fmt.Sprintf("skip make torrent: %s \n", item.Type+"_"+item.ID))
			log.Printf("skip make torrent: %s", item.Type+"_"+item.ID)
		}

		log.Printf("++load to bt client: %s", item.Type+"_"+item.ID)
		task.Writer.Write(fmt.Sprintf("++load to bt client: %s \n", item.Type+"_"+item.ID))
		go func(path, torrentPath, typee, id string) {
			defer wg.Done()
			if err := Download(mg.BTClient, path, torrentPath, config); err != nil {
				log.Printf("download err: %s", err.Error())
				return
			}
			log.Printf("--load to bt client: %s", typee+"_"+id)
			task.Writer.Write(fmt.Sprintf("--load to bt client: %s \n", typee+"_"+id))
		}(packagePath, torrentPath, item.Type, item.ID)
	}

	wg.Wait()

	return
}

func prepareForMultiLayer(mg *Manager, task *Task) (err error) {
	ids, err := p2p.GetLayerIDs(mg.DockerClient, task.ImageName)
	if err != nil {
		return
	}

	task.History = ids

	allExist := true

	for _, layerId := range ids {
		id := task.ImageID + "_" + layerId
		task.Items = append(task.Items, &p2p.Item{
			ID:   id,
			Type: "multi_layer",
			URL:  mg.FileServerPrefix + "multi_layer/" + id + ".torrent",
		})

		exist, _, err := mg.PackageExist(id, "multi_layer")
		if err != nil {
			return err
		}

		if allExist && !exist {
			allExist = false
		}
	}

	if !allExist {
		dir := filepath.Join(os.TempDir(), strconv.Itoa(int(time.Now().Unix())))
		if err = os.Mkdir(dir, 644); err != nil {
			return err
		}
		defer os.RemoveAll(dir)

		path := filepath.Join(dir, task.ImageID+".tar")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		log.Printf("++save image: %s", task.ImageName)
		task.Writer.Write(fmt.Sprintf("++save image: %s \n", task.ImageName))
		if err = Save(mg.DockerClient, task.ImageName, f); err != nil {
			return err
		}
		if err = f.Sync(); err != nil {
			return err
		}
		task.Writer.Write(fmt.Sprintf("--save image: %s \n", task.ImageName))
		log.Printf("--save image: %s", task.ImageName)

		for _, item := range task.Items {
			packageExist, packagePath, err := mg.PackageExist(item.ID, "multi_layer")
			if err != nil {
				return err
			}

			if packageExist {
				task.Writer.Write(fmt.Sprintf("skip extract layer: %s \n", item.ID))
				log.Printf("skip extract layer: %s", item.ID)
				continue
			}

			log.Printf("++extract layer: %s", item.ID)
			task.Writer.Write(fmt.Sprintf("++extract layer: %s \n", item.ID))

			endId := strings.Split(item.ID, "_")[1]
			extractIds := make(map[string]string)

			for _, id := range ids {
				extractIds[id] = ""
				if id == endId {
					break
				}
			}

			pf, err := os.Create(packagePath)
			if err != nil {
				return err
			}

			gw := gzip.NewWriter(pf)

			tw := tar.NewWriter(gw)

			if _, err = f.Seek(0, 0); err != nil {
				return err
			}

			tr := tar.NewReader(f)

			for {
				header, err := tr.Next()
				if err != nil {
					if err == io.EOF {
						break
					}

					return err
				}

				if header.Typeflag != tar.TypeDir {
					continue
				}

				id := header.Name[:len(header.Name)-1]

				if _, ok := extractIds[id]; !ok {
					continue
				}

				if err = tw.WriteHeader(header); err != nil {
					return err
				}

				for i := 0; i < 3; i++ {
					header, err := tr.Next()
					if err != nil {
						return err
					}

					if err = tw.WriteHeader(header); err != nil {
						return err
					}

					if _, err = io.Copy(tw, tr); err != nil {
						return err
					}
				}
			}

			tw.Close()
			gw.Close()
			pf.Close()

			task.Writer.Write(fmt.Sprintf("--extract layer: %s \n", item.ID))
			log.Printf("--extract layer: %s", item.ID)
		}

	}

	var wg sync.WaitGroup
	wg.Add(len(task.Items))

	config := make(map[string]string)
	config["target"] = "manager"
	for _, item := range task.Items {
		torrentExist, torrentPath, err := mg.TorrentExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		packageExist, packagePath, err := mg.PackageExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		if !torrentExist {

			if !packageExist {
				return fmt.Errorf("[ERROR]package not exist: %s", packagePath)
			}

			log.Printf("++make torrent: %s", item.Type+"_"+item.ID)
			task.Writer.Write(fmt.Sprintf("++make torrent: %s \n", item.Type+"_"+item.ID))
			if err = CreateTorrent(mg.BTClient, packagePath, torrentPath, mg.Trackers); err != nil {
				return err
			}
			task.Writer.Write(fmt.Sprintf("--make torrent: %s \n", item.Type+"_"+item.ID))
			log.Printf("--make torrent: %s", item.Type+"_"+item.ID)

		} else {
			task.Writer.Write(fmt.Sprintf("skip make torrent: %s \n", item.Type+"_"+item.ID))
			log.Printf("skip make torrent: %s", item.Type+"_"+item.ID)
		}

		log.Printf("++load to bt client: %s", item.Type+"_"+item.ID)
		task.Writer.Write(fmt.Sprintf("++load to bt client: %s \n", item.Type+"_"+item.ID))
		go func(path, torrentPath, typee, id string) {
			defer wg.Done()
			if err := Download(mg.BTClient, path, torrentPath, config); err != nil {
				log.Printf("download err: %s", err.Error())
				return
			}
			log.Printf("--load to bt client: %s", typee+"_"+id)
			task.Writer.Write(fmt.Sprintf("--load to bt client: %s \n", typee+"_"+id))
		}(packagePath, torrentPath, item.Type, item.ID)
	}

	wg.Wait()

	return
}

func Pull(client *docker.Client, image, username, password, email string) (err error) {
	if err = p2p.PullImage(client, image, username, password, email); err != nil {
		return
	}
	return
}

func Save(client *docker.Client, image string, w io.Writer) (err error) {
	if err = p2p.SaveImage(client, image, w); err != nil {
		return err
	}
	return
}

func CreateTorrent(bt bittorrent.BitTorrent, path, torrentPath string, trackers []string) (err error) {
	if bt.CreateTorrent(path, torrentPath, trackers); err != nil {
		return
	}
	return
}

func Download(client bittorrent.BitTorrent, path, torrentPath string, config map[string]string) (err error) {
	if err = client.Download(path, torrentPath, config); err != nil {
		return
	}
	return
}
