package p2p_lib

import (
	"fmt"
	docker "github.com/fsouza/go-dockerclient"
	"log"
	"os"
)

func PullImage(client *docker.Client, image, username, password, email string) error {
	opts := docker.PullImageOptions{
		Repository: image,
		//OutputStream: os.Stdout,
	}

	auth := docker.AuthConfiguration{
		Username: username,
		Password: password,
		Email:    email,
	}

	if err := client.PullImage(opts, auth); err != nil {
		return err
	}

	return nil
}

func SaveImage(client *docker.Client, image, path string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	opts := docker.ExportImageOptions{
		Name:         image,
		OutputStream: f,
	}
	if err := client.ExportImage(opts); err != nil {
		return err
	}

	return nil
}

func GetImageIDByImageName(client *docker.Client, image string) (string, error) {
	im, err := client.InspectImage(image)
	if err != nil {
		return "", err
	}

	return im.ID, nil
}

func LoadImage(client *docker.Client, path string) (err error) {

	log.Printf("starting load image %s", path)

	defer func() {
		if err != nil {
			err = fmt.Errorf("load image error: %s", err.Error())
		} else {
			log.Printf("complete load image %s", path)
		}
	}()

	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	opts := docker.LoadImageOptions{
		InputStream: file,
	}

	err = client.LoadImage(opts)

	return
}

func ImageExist(client *docker.Client, name, id string) (ex bool, err error) {
	image, err := client.InspectImage(name)
	if err != nil {
		if err == docker.ErrNoSuchImage {
			return false, nil
		}
		return false, err
	}

	return image.ID == id, nil
}
