package manager

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"p2p_lib"
	"p2p_lib/util"
	"path/filepath"
	"strings"
)

type tagId map[string]string
type repositories map[string]tagId

func Pull(mg *Manager, task *Task) (err error) {

}

func Save(mg *Manager, task *Task) (r io.Reader, err error) {

}

func UnTar(r io.Reader) (err error) {

}

func Compress(r io.Reader) (err error) {

}

func MakeTorrent() (err error) {

}

func Prepare(manager *Manager, task *DistributionTask) (err error) {
	defer func() {
		if err != nil {
			//TODO process auth error
			fmt.Errorf("prepare error: %s", err.Error())
		}
	}()

	task.Client.Write("pulling image")

	log.Printf("++++pull: %s", task.ImageName)
	if err = p2p_lib.PullImage(manager.DockerClient, task.ImageName, task.Username, task.Password, task.Email); err != nil {
		return
	}
	log.Printf("--------pull: %s", task.ImageName)

	id, err := p2p_lib.GetImageIDByImageName(manager.DockerClient, task.ImageName)
	if err != nil {
		return
	}

	task.ImageID = id
	task.PD.ImageID = id

	if task.Mode == p2p_lib.ImageMode {
		err = prepareImageMode(manager, task)
	} else {
		err = prepareLayerMode(manager, task)
	}

	return
}

func prepareImageMode(manager *Manager, task *DistributionTask) (err error) {
	packageExist, packagePath, err := manager.PackageExist(task.ImageID, task.Mode)
	if err != nil {
		return
	}

	//TODO do not write image.tar to disk
	if !packageExist {
		log.Printf("++++compress: %s %s", task.ImageName, packagePath)
		imageTarExist, imageTarPath, err := manager.ImageTarExist(task.ImageID)
		if err != nil {
			return err
		}

		if !imageTarExist {
			log.Printf("++++export: %s %s", task.ImageName, imageTarPath)
			if err = p2p_lib.SaveImage(manager.DockerClient, task.ImageName, imageTarPath); err != nil {
				return err
			}
			log.Printf("--------export: %s %s", task.ImageName, imageTarPath)
		} else {
			log.Printf("skip export: %s %s", task.ImageName, imageTarPath)
		}

		if err = util.Gzip(imageTarPath, packagePath); err != nil {
			return err
		}
		log.Printf("--------compress: %s %s", task.ImageName, packagePath)
	} else {
		log.Printf("skip compress: %s %s", task.ImageName, packagePath)
	}

	torrentExist, torrentPath, err := manager.TorrentExist(task.ImageID, task.Mode)
	if err != nil {
		return
	}

	if !torrentExist {
		log.Printf("++++make torrent: %s %s", packagePath, torrentPath)
		if err = manager.BTClient.CreateTorrent(packagePath, torrentPath, manager.Trackers); err != nil {
			return
		}
		log.Printf("--------make torrent: %s %s", packagePath, torrentPath)
	} else {
		log.Printf("skip make torrent: %s %s", packagePath, torrentPath)
	}

	task.Torrents[packagePath] = torrentPath

	//	taskExist, taskPath, err := manager.TaskExist(task.ImageID, task.Mode)
	//	if err != nil {
	//		return
	//	}

	//	if !taskExist {
	//		log.Printf("++++create task file: %s %s", task.ImageName, taskPath)
	//		if err = util.Copy(torrentPath, taskPath[:strings.LastIndex(taskPath, string(filepath.Separator))]); err != nil {
	//			return
	//		}
	//		log.Printf("--------create task file: %s %s", task.ImageName, taskPath)
	//	} else {
	//		log.Printf("skip create task file: %s %s", task.ImageName, taskPath)
	//	}

	//	task.TaskPath = taskPath

	//TODO port
	task.PD.URLs = append(task.PD.URLs, "http://"+manager.IPOrHostname+":8000/torrent/image_"+task.ImageID+".torrent")

	return
}

//TODO v1 v2
func prepareLayerMode(manager *Manager, task *DistributionTask) (err error) {
	imageTarExist, imageTarPath, err := manager.ImageTarExist(task.ImageID)
	if err != nil {
		return
	}

	//TODO do not write image.tar to harddisk
	if !imageTarExist {
		log.Printf("++++export: %s %s", task.ImageName, imageTarPath)
		err = p2p_lib.SaveImage(manager.DockerClient, task.ImageName, imageTarPath)
		if err != nil {
			return
		}
		log.Printf("--------export %s %s", task.ImageName, imageTarPath)
	} else {
		log.Printf("skip export: %s  %s", task.ImageName, imageTarPath)
	}

	//paths of torrent and metadata files, all of these files will be tared and distributed to hosts
	var paths []string

	//	//key: path of layerID.torrent value: path of layerID.tar.gz
	//	var torrents []string

	imageFile, err := os.Open(imageTarPath)
	if err != nil {
		return
	}
	defer imageFile.Close()

	tarReader := tar.NewReader(imageFile)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			id := strings.TrimRight(header.Name, string(filepath.Separator))
			layerExist, layerPath, err := manager.PackageExist(id, task.Mode)
			if err != nil {
				return err
			}
			if !layerExist {
				log.Printf("++++extract: %s %s", task.ImageName, id)
				layerFile, err := os.Create(layerPath)
				if err != nil {
					return err
				}
				defer layerFile.Close()

				gw := gzip.NewWriter(layerFile)

				tw := tar.NewWriter(gw)

				for i := 0; i < 3; i++ {
					header, err = tarReader.Next()
					if err != nil {
						return err
					}

					if err = tw.WriteHeader(header); err != nil {
						return err
					}

					if _, err = io.Copy(tw, tarReader); err != nil {
						return err
					}
				}

				tw.Close()
				gw.Close()

				layerFile.Close()

				log.Printf("--------extract: %s %s", task.ImageName, id)

			} else {
				log.Printf("skip extract: %s %s", task.ImageName, id)
			}

			layerTorrentExist, layerTorrentPath, err := manager.TorrentExist(id, task.Mode)
			if err != nil {
				return err
			}

			if !layerTorrentExist {
				log.Printf("++++make torrent: %s %s", id, layerTorrentPath)
				if err := manager.BTClient.CreateTorrent(layerPath, layerTorrentPath, manager.Trackers); err != nil {
					return err
				}
				log.Printf("--------make torrent: %s %s", id, layerTorrentPath)
			} else {
				log.Printf("skip make torrent: %s %s", id, layerTorrentPath)
			}
			paths = append(paths, layerTorrentPath)
			task.Torrents[layerPath] = layerTorrentPath

		case tar.TypeReg:
			if header.Name == "repositories" {
				log.Printf("++++extract: %s", header.Name)

				data := &bytes.Buffer{}

				if _, err = io.Copy(data, tarReader); err != nil {
					return err
				}

				var repos repositories

				if err := json.Unmarshal(data.Bytes(), &repos); err != nil {
					return err
				}

				var id string

				for _, v := range repos {
					for _, v1 := range v {
						id = v1
						break
					}
					break
				}

				metadataExist, metadataPath, err := manager.MetadataExist(id)
				if err != nil {
					return err
				}

				if !metadataExist {
					metadataFile, err := os.Create(metadataPath)
					if err != nil {
						return err
					}
					defer metadataFile.Close()

					if _, err = io.Copy(metadataFile, data); err != nil {
						return err
					}
					paths = append(paths, metadataPath)
					log.Printf("--------extract: %s", header.Name, metadataPath)
				} else {
					log.Printf("skip extract: %s", header.Name, metadataPath)
				}
			}

		default:
			return fmt.Errorf("unsupport type flag")
		}
	}

	exist, taskPath, err := manager.TaskExist(task.ImageID, task.Mode)
	if err != nil {
		return
	}

	if !exist {
		log.Printf("++++create task tar: %s", task.ImageName)

		taskFile, err := os.Create(taskPath)
		if err != nil {
			return err
		}
		defer taskFile.Close()

		gw := gzip.NewWriter(taskFile)
		defer gw.Close()

		tw := tar.NewWriter(gw)
		defer tw.Close()

		for _, p := range paths {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}

		log.Printf("--------create task tar: %s", task.ImageName)
	} else {
		log.Printf("skip create task tar: %s", task.ImageName)
	}

	task.TaskPath = taskPath

	return
}