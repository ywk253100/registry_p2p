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
	"strings"
	"sync"

	docker "github.com/fsouza/go-dockerclient"
)

type tagId map[string]string
type repositories map[string]tagId

func Prepare(mg *Manager, task *Task) (err error) {

	kind := task.Mode
image:
	c, err := mg.PoolAdd(kind + "_" + task.ImageName)
	if err != nil {
		if c != nil {
			log.Printf("task is already in progress: %s", kind+"_"+task.ImageName)
			task.Writer.Write(fmt.Sprintf("task is already in progress: %s", kind+"_"+task.ImageName))
			<-c
			goto image
		} else {
			return err
		}
	}
	defer mg.PoolDelete(kind + "_" + task.ImageName)

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

	if task.Mode == "image" {
		task.Items = append(task.Items, &p2p.Item{
			ID:   task.ImageID,
			Type: "image",
			URL:  mg.FileServerPrefix + "image_" + task.ImageID + ".torrent",
		})
	} else {
		ids, err := p2p.GetLayerIDs(mg.DockerClient, task.ImageName)
		if err != nil {
			return err
		}
		for _, id := range ids {
			task.Items = append(task.Items, &p2p.Item{
				ID:   id,
				Type: "layer",
				URL:  mg.FileServerPrefix + "layer_" + id + ".torrent",
			})
		}
		task.Items = append(task.Items, &p2p.Item{
			ID:   task.ImageID,
			Type: "metadata",
			URL:  mg.FileServerPrefix + "metadata_" + task.ImageID + ".torrent",
		})
	}

	var wg sync.WaitGroup
	wg.Add(len(task.Items))

	for _, item := range task.Items {
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

				log.Printf("++save: %s", task.ImageName)
				task.Writer.Write(fmt.Sprintf("++save: %s\n", task.ImageName))
				if err = Save(mg.DockerClient, task.ImageName, imageTarFile); err != nil {
					os.Remove(imageTarPath)
					return err
				}
				log.Printf("--save: %s", task.ImageName)
				task.Writer.Write(fmt.Sprintf("--save: %s\n", task.ImageName))

				if _, err = imageTarFile.Seek(0, 0); err != nil {
					return err
				}
			} else {
				log.Printf("image tar exist: %s", task.ImageName)
				task.Writer.Write(fmt.Sprintf("image tar exist: %s", task.ImageName))
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

				log.Printf("++compress: %s", task.ImageName)
				task.Writer.Write(fmt.Sprintf("++compress: %s\n", task.ImageName))
				if err = TarCompress(imageTarFile, nil, packageFile, item.Type); err != nil {
					os.Remove(packagePath)
					return err
				}
				log.Printf("--compress: %s", task.ImageName)
				task.Writer.Write(fmt.Sprintf("--compress: %s\n", task.ImageName))

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

					layer:
						c, err := mg.PoolAdd("layer_" + id)
						if err != nil {
							if c != nil {
								<-c
								log.Printf("task is already in progress: %s", "layer_"+id)
								task.Writer.Write(fmt.Sprintf("task is already in progress: %s", "layer_"+id))
								goto layer
							} else {
								return err
							}
						}
						defer mg.PoolDelete("layer_" + id)

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

						log.Printf("++compress: layer %s", id)
						task.Writer.Write(fmt.Sprintf("++compress: layer %s\n", id))
						if err = TarCompress(tr, nil, packageFile, "layer"); err != nil {
							os.Remove(packagePath)
							return err
						}
						log.Printf("--compress: layer %s", id)
						task.Writer.Write(fmt.Sprintf("--compress: layer %s\n", id))

					case tar.TypeReg:
						if header.Name == "repositories" {
						metadata:
							c, err := mg.PoolAdd("metadata_" + id)
							if err != nil {
								if c != nil {
									<-c
									log.Printf("task is already in progress: %s", "metadata_"+id)
									task.Writer.Write(fmt.Sprintf("task is already in progress: %s", "metadata_"+id))
									goto metadata
								} else {
									return err
								}
							}
							defer mg.PoolDelete("metadata_" + id)

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

							log.Printf("++compress: metadata %s", id)
							task.Writer.Write(fmt.Sprintf("++compress: metadata %s\n", id))
							if err = TarCompress(tr, header, packageFile, "metadata"); err != nil {
								os.Remove(packagePath)
								return err
							}
							log.Printf("--compress: metadata %s", id)
							task.Writer.Write(fmt.Sprintf("--compress: metadata %s\n", id))

						}

					default:
						return fmt.Errorf("unsupported type flag")
					}
				}
			}
		} else {
			log.Printf("package exist: %s", item.Type+"_"+item.ID)
			task.Writer.Write(fmt.Sprintf("package exist: %s\n", item.Type+"_"+item.ID))
		}

		torrentExist, torrentPath, err := mg.TorrentExist(item.ID, item.Type)
		if err != nil {
			return err
		}

		if !torrentExist {
			log.Printf("++make torrent: %s", item.Type+"_"+item.ID)
			task.Writer.Write(fmt.Sprintf("++make torrent: %s\n", item.Type+"_"+item.ID))
			if err = CreateTorrent(mg.BTClient, packagePath, torrentPath, mg.Trackers); err != nil {
				os.Remove(torrentPath)
				return err
			}
			log.Printf("--make torrent: %s", item.Type+"_"+item.ID)
			task.Writer.Write(fmt.Sprintf("--make torrent: %s\n", item.Type+"_"+item.ID))
		} else {
			log.Printf("torrent exist: %s", item.Type+"_"+item.ID)
			task.Writer.Write(fmt.Sprintf("torrent exist: %s\n", item.Type+"_"+item.ID))
		}
		config := make(map[string]string)
		config["target"] = "manager"

		go func(path, torrentPath string) {
			defer wg.Done()
			if err := Download(mg.BTClient, path, torrentPath, config); err != nil {
				log.Printf("download err: %s", err.Error())
				return
			}
		}(packagePath, torrentPath)
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
