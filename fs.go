package docker

import (
	"fmt"
	"io"

	"github.com/moby/moby/api/types"
	"github.com/pkg/errors"
)

func (c *Container) CopyToContainer(targetPath string, content io.Reader) error {
	if err := c.checkIsRunning(); err != nil {
		return err
	}
	client := c.client
	err := client.CopyToContainer(
		c.options.context,
		c.ID,
		targetPath,
		content,
		types.CopyToContainerOptions{
			AllowOverwriteDirWithFile: true,
		},
	)
	if err != nil {
		msg := fmt.Sprintf("Failed to copy to container at %s", targetPath)
		log.WithError(err).
			WithField("target", targetPath).
			Error(msg)
		return errors.Wrapf(err, msg)
	}
	log.WithField("target", targetPath).
		Debug("copied content to container successfully")
	return nil
}

func (c *Container) CopyFromContainer(sourcePath string) (io.ReadCloser, error) {
	if err := c.checkIsRunning(); err != nil {
		return nil, err
	}

	client := c.client

	rc, _, err := client.CopyFromContainer(
		c.options.context,
		c.ID,
		sourcePath,
	)
	if err != nil {
		msg := fmt.Sprintf("Failed to download dir = %s from container to the host",
			sourcePath)
		log.WithError(err).
			WithField("source", sourcePath).
			Error(msg)
		return nil, errors.Wrapf(err, msg)
	}
	log.WithField("source", sourcePath).
		Debug("downloaded from container successfully")
	return rc, nil
}

func (c *Container) checkIsRunning() error {
	client := c.client
	info, err := client.ContainerInspect(
		c.options.context,
		c.ID,
	)
	if err != nil {
		msg := "Failed to inspect container."
		log.WithError(err).WithField("container_id", c.ID).Error(msg)
		return errors.Wrapf(err, msg+" container_id = %s", c.ID)
	}
	if !info.State.Running {
		msg := "Expecting container to be running, but was not."
		log.WithField("container_id", c.ID).Error(msg)
		return errors.Wrapf(err, msg+" container_id = %s", c.ID)
	}

	return nil
}
