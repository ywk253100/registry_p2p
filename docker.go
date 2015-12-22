package registry_p2p

import (
	docker "github.com/fsouza/go-dockerclient"
	"io"
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

func SaveImage(client *docker.Client, image string, w io.Writer) (err error) {
	opts := docker.ExportImageOptions{
		Name:         image,
		OutputStream: w,
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

func LoadImage(client *docker.Client, r io.Reader) (err error) {
	opts := docker.LoadImageOptions{
		InputStream: r,
	}

	if err = client.LoadImage(opts); err != nil {
		return
	}

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

func GetLayerIDs(client *docker.Client, name string) (ids []string, err error) {
	histories, err := client.ImageHistory(name)
	if err != nil {
		return
	}
	for _, history := range histories {
		ids = append(ids, history.ID)
	}
	return
}

func Inspect(client *docker.Client, name string) (image *docker.Image, err error) {
	image, err = client.InspectImage(name)
	return
}

func GetParentID(client *docker.Client, id string) (parentID string, err error) {
	image, err := Inspect(client, id)
	if err == nil {
		parentID = image.Parent
	}
	return
}

func Tag(client *docker.Client, id, repo, tag string) (err error) {
	opts := docker.TagImageOptions{
		Repo: repo,
		Tag:  tag,
	}

	err = client.TagImage(id, opts)
	return
}
