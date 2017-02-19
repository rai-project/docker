package docker

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
)

func (c *Client) ListImages(image, tag string) ([]docker.APIImages, error) {
	client := c.Client
	imgs, err := client.ListImages(docker.ListImagesOptions{
		All:    false,
		Filter: fmt.Sprintf("%s:%s", image, tag),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of images from docker")
	}
	return imgs, nil
}

func (c *Client) PullImage(image, tag string) error {
	client := c.Client
	log.WithField("repository", image).
		WithField("tag", tag).
		Debug("Pulling docker image")
	err := client.PullImage(
		docker.PullImageOptions{
			Repository:   image,
			Tag:          tag,
			OutputStream: log.Logger.Out,
		},
		docker.AuthConfiguration{},
	)
	if err != nil {
		err = Error(err)
		log.WithError(err).
			WithField("repository", image).
			WithField("tag", tag).
			Error("Not able to pull docker image")
		return err
	}
	log.WithField("repository", image).
		WithField("tag", tag).
		Debug("Pulled docker image")
	return nil
}

func (c *Client) GetImage(image, tag string) error {
	imgs, err := c.ListImages(image, tag)
	if err != nil {
		return err
	}
	if len(imgs) == 0 || tag == "lastest" {
		return c.PullImage(image, tag)
	}
	return nil
}
