package manager

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	p2p "registry_p2p"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

type Item struct {
	ID   string
	Type string
}

type tagId map[string]string
type repositories map[string]tagId

func Prepare(mg *Manager, task *Task) (err error) {
	kind := task.Mode
	c, err := mg.PoolAdd(kind + "_" + task.ImageName)
	if err != nil {
		if c != nil {
			<-c
			goto goon1
		} else {
			return err
		}
	}

	task.State = "pulling"
	if err = Pull(mg.DockerClient, task.ImageName, task.Username, task.Password, task.Email); err != nil {
		return
	}

	id, err := p2p.GetImageIDByImageName(mg.DockerClient, task.ImageName)
	if err != nil {
		return
	}

	task.ImageID = id

	var items []*Item

	if task.Mode == "image" {
		items = append(items, &Item{
			ID:   task.ImageID,
			Type: "image",
		})
	} else {
		ids, err := p2p.GetLayerIDs(mg.DockerClient, task.ImageName)
		if err != nil {
			return err
		}
		for _, id := range ids {
			items = append(items, &Item{
				ID:   id,
				Type: "layer",
			})
		}
		items = append(items, &Item{
			ID:   task.ImageID,
			Type: "metadata",
		})
	}

	for _, item := range items {
		packageExist, packagePath, err := mg.PackageExist(item.ID, item.Type)
		if err != nil {
			return err
		}
		if !packageExist {
			imageTarExist, imageTarPath, err := mg.ImageTarExist(task.ImageID)
			if err != nil {
				return err
			}
			var imageTarFile *os.File
			if !imageTarExist {
				imageTarFile, err = os.Create(imageTarPath)
				if err != nil {
					return err
				}
				defer imageTarFile.Close()
				if err = Save(mg.DockerClient, task.ImageName, imageTarFile); err != nil {
					return err
				}
				if _, err = imageTarFile.Seek(0, 0); err != nil {
					return err
				}
			} else {
				imageTarFile, err = os.Open(imageTarPath)
				if err != nil {
					return err
				}
				defer imageTarFile.Close()
			}

			if item.Type == "image" {
				packageFile, err := os.Create(packagePath)
				if err != nil {
					return err
				}
				defer packageFile.Close()
				if err = TarCompress(imageTarFile, packageFile, item.Type); err != nil {
					return err
				}
			} else {
				tr := tar.NewReader(imageTarFile)
				for {
					header, err := tr.Next()
					if err != nil {
						if err == io.EOF {
							break
						}
						return err
					}

					switch header.Typeflag {
					case tar.TypeDir:
						id := strings.TrimRight(header.Name, string(filepath.Separator))

						c, err := mg.PoolAdd("layer_" + id)
						if err != nil {
							if c != nil {
								<-c
								continue
							} else {
								return err
							}
						}

						packageExist, packagePath, err := mg.PackageExist(id, "layer")
						if err != nil {
							return err
						}
						if packageExist {
							continue
						}

						packageFile, err := os.Create(packagePath)
						if err != nil {
							return err
						}
						defer packageFile.Close()

						if err = TarCompress(tr, packageFile, "layer"); err != nil {
							return err
						}

						if err = mg.PoolDelete("metadata_" + id); err != nil {
							return err
						}

					case tar.TypeReg:
						if header.Name == "repositories" {
							c, err := mg.PoolAdd("layer_" + id)
							if err != nil {
								if c != nil {
									<-c
									continue
								} else {
									return err
								}
							}

							packageExist, packagePath, err := mg.PackageExist(task.ImageID, "metadata")
							if err != nil {
								return err
							}
							if packageExist {
								continue
							}

							packageFile, err := os.Create(packagePath)
							if err != nil {
								return err
							}
							defer packageFile.Close()

							if err = TarCompress(tr, packageFile, "metadata"); err != nil {
								return err
							}
							if err = mg.PoolDelete("metadata_" + id); err != nil {
								return err
							}
						}

					default:
						return fmt.Errorf("unsupported type flag")
					}
				}
				break
			}
		}
	}

	for _, item := range items {
		torrentExist, torrentPath, err := mg.TorrentExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		if !torrentExist {
			//make torrent
		}
	}

	if err = mg.PoolDelete(kind + "_" + id); err != nil {
		return err
	}

goon1:
	var agts []*AgentTaskItem

	for _, item := range items {
		ati := &AgentTaskItem{
			Type: item.Type,
			URL:  mg.FileServerPrefix + item.Type + "_" + item.ID + ".torrent",
		}
		agts = append(agts, ati)
	}

	for _, host := range task.Hosts {
		task.AgentTasks = append(task.AgentTasks, &AgentTask{
			ID:    host,
			Items: agts,
		})
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

func TarCompress(r io.Reader, w io.Writer, typee string) (err error) {
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
		tr, _ := tar.Reader.(r)

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
		tr := tar.Reader(r)
		header, err := tr.Next()
		if err != nil {
			return err
		}
		if err = tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err = io.Copy(tw, tarReader); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type: %s", typee)
	}

	return
}

func MakeTorrent() (err error) {
	return
}

//func Prepare(manager *Manager, task *Task) (err error) {
//	defer func() {
//		if err != nil {
//			//TODO process auth error
//			fmt.Errorf("prepare error: %s", err.Error())
//		}
//	}()

//	task.Client.Write("pulling image")

//	log.Printf("++++pull: %s", task.ImageName)
//	if err = p2p_lib.PullImage(manager.DockerClient, task.ImageName, task.Username, task.Password, task.Email); err != nil {
//		return
//	}
//	log.Printf("--------pull: %s", task.ImageName)

//	id, err := p2p_lib.GetImageIDByImageName(manager.DockerClient, task.ImageName)
//	if err != nil {
//		return
//	}

//	task.ImageID = id
//	task.PD.ImageID = id

//	if task.Mode == p2p_lib.ImageMode {
//		err = prepareImageMode(manager, task)
//	} else {
//		err = prepareLayerMode(manager, task)
//	}

//	return
//}

/*
func prepareImageMode(manager *Manager, task *Task) (err error) {
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
			if err = p2p.SaveImage(manager.DockerClient, task.ImageName, imageTarPath); err != nil {
				return err
			}
			log.Printf("--------export: %s %s", task.ImageName, imageTarPath)
		} else {
			log.Printf("skip export: %s %s", task.ImageName, imageTarPath)
		}

		if err = utils.Gzip(imageTarPath, packagePath); err != nil {
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
func prepareLayerMode(manager *Manager, task *Task) (err error) {
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
					tarReader.
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
*/
