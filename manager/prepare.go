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

	if task.Mode == p2p.MODE_IMAGE {
		return prepareForImage(mg, task)
	} else {
		return prepareForLayer(mg, task)
	}

	return
}

func prepareForImage(mg *Manager, task *Task) (err error) {

	task.URL = mg.FileServerPrefix + "image_" + task.ImageID + ".torrent"

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
	parentID, err := p2p.GetParentID(mg.DockerClient, id)

	task.Items = append(task.Items, &p2p.Item{
		ID:       id,
		ParentID: parentID,
		Type:     "layer_meta",
		URL:      mg.FileServerPrefix + "layer_meta_" + id + ".torrent",
	})

	for len(parentID) != 0 {
		id = parentID
		parentID, err = p2p.GetParentID(mg.DockerClient, id)

		task.Items = append(task.Items, &p2p.Item{
			ID:       id,
			ParentID: parentID,
			Type:     "layer",
			URL:      mg.FileServerPrefix + "layer_" + id + ".torrent",
		})
	}

	pExist := true

	for _, item := range task.Items {
		packageExist, _, err := mg.PackageExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		if !packageExist {
			pExist = false
			break
		}
	}

	if !pExist {

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

		log.Printf("++save image: %s", task.ImageName)
		task.Writer.Write(fmt.Sprintf("++save image: %s \n", task.ImageName))
		if err = Save(mg.DockerClient, task.ImageName, f); err != nil {
			return err
		}
		f.Close()
		task.Writer.Write(fmt.Sprintf("--save image: %s \n", task.ImageName))
		log.Printf("--save image: %s", task.ImageName)

		for _, item := range task.Items {
		layer:
			packageExist, packagePath, err := mg.PackageExist(item.ID, item.Type)
			if err != nil {
				return err
			}

			if !packageExist {
				c, err := mg.PoolAdd(item.Type + "_" + item.ID)
				if err != nil {
					if c != nil {
						log.Printf("layer is being extract by other progress, please wait: %s", task.ImageName)
						task.Writer.Write(fmt.Sprintf("layer is being extract by other progress, please wait: %s \n", task.ImageName))
						<-c
						goto layer
					} else {
						return err
					}
				}
				defer mg.PoolDelete(item.Type + "_" + item.ID)

				log.Printf("++extract layer: %s", item.Type+"_"+item.ID)
				task.Writer.Write(fmt.Sprintf("++extract layer: %s \n", item.Type+"_"+item.ID))
				if err = extract(path, packagePath, item.ID, item.Type); err != nil {
					return err
				}
				task.Writer.Write(fmt.Sprintf("--extract layer: %s \n", item.Type+"_"+item.ID))
				log.Printf("--extract layer: %s", item.Type+"_"+item.ID)
			} else {
				task.Writer.Write(fmt.Sprintf("skip extract layer: %s \n", item.Type+"_"+item.ID))
				log.Printf("skip extract layer: %s", item.Type+"_"+item.ID)
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

func extract(tarPath, packagePath, id, typee string) (err error) {
	pf, err := os.Create(packagePath)
	if err != nil {
		return err
	}
	defer func() {
		pf.Close()
		if err != nil {
			os.Remove(packagePath)
		}
	}()

	gw := gzip.NewWriter(pf)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	tf, err := os.Open(tarPath)
	if err != nil {
		return
	}
	defer tf.Close()

	tr := tar.NewReader(tf)

	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		if header.Name[:len(header.Name)-1] == id {
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

			//layer_meta
			if typee == "layer_meta" {
				for {
					header, err := tr.Next()
					if err != nil {
						return err
					}

					if header.Name == "repositories" {
						if err = tw.WriteHeader(header); err != nil {
							return err
						}

						if _, err = io.Copy(tw, tr); err != nil {
							return err
						}

						break
					}
				}
			}
		}
	}
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

func TarCompress(r io.Reader, header *tar.Header, w io.Writer, typee string) (err error) {
	gw := gzip.NewWriter(w)
	defer gw.Close()
	switch typee {
	case "image":
		if _, err = io.Copy(gw, r); err != nil {
			return
		}
		return
	case "layer":
		tw := tar.NewWriter(gw)
		defer tw.Close()
		tr, ok := r.(*tar.Reader)
		if !ok {
			return errors.New("r is not the tar.Reader type")
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
	case "metadata":
		tw := tar.NewWriter(gw)
		defer tw.Close()
		tr, ok := r.(*tar.Reader)
		if !ok {
			return errors.New("r is not the tar.Reader type")
		}
		if err != nil {
			return err
		}
		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err = io.Copy(tw, tr); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type: %s", typee)
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
