package docker

import (
	"fmt"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"bitbucket.org/hwuligans/rai/pkg/config"
	"bitbucket.org/hwuligans/rai/pkg/uuid"
)

type Container struct {
	ImageName string
	Tag       string
	Context   context.Context
	client    *Client
	cancel    context.CancelFunc
	*dc.Container
}

func NewContainer(client *Client) (*Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.Docker.TimeLimit)
	env := baseEnvsStringList()
	name := fmt.Sprintf("%s-run-%s", config.App.Name, uuid.NewV4())
	opts := dc.CreateContainerOptions{
		Name:       name,
		HostConfig: hostConfig,
		Config: &dc.Config{
			Image:           fmt.Sprintf("%s:%s", config.Docker.Image, config.Docker.Tag),
			Hostname:        name,
			User:            config.Docker.User,
			Env:             env,
			Memory:          config.Docker.MemoryLimit,
			WorkingDir:      "/build",
			AttachStdin:     false,
			AttachStdout:    true,
			AttachStderr:    true,
			OpenStdin:       true,
			StdinOnce:       true,
			Tty:             true,
			NetworkDisabled: true,
			VolumeDriver:    "nvidia-docker",
			Mounts: []dc.Mount{
				dc.Mount{
					Name:        "nvidia_driver_367.57",
					Source:      "/var/lib/nvidia-docker/volumes/nvidia_driver/367.57",
					Destination: "/usr/local/nvidia",
					Driver:      "nvidia-docker",
					Mode:        "ro",
					RW:          false,
				},
			},
			Cmd: []string{
				"sleep",
				"4h",
			},
			// Cmd: []string{
			// 	"/bin/sh",
			// 	"-c",
			// 	"while true; do date >> /tmp/date.log; sleep 1; done",
			// },
		},
		Context: ctx,
	}
	cont, err := client.CreateContainer(opts)
	if err != nil {
		err = Error(err)
		log.WithError(err).Error("Failed to create docker container.")
		return nil, errors.Wrap(err, "Failed to create docker container.")
	}
	res := &Container{
		ImageName: config.Docker.Image,
		Tag:       config.Docker.Tag,
		Container: cont,
		client:    client,
		cancel:    cancel,
		Context:   ctx,
	}
	go func() {
		<-ctx.Done()
	}()
	return res, nil
}

func (c *Container) Start() error {
	return c.client.StartContainer(c.ID, hostConfig)
}

func (c *Container) Stop() error {
	defer func() {
		log.WithField("container", c.ID).Debug("stopping container")
		c.cancel()
	}()
	err := c.client.StopContainer(c.ID, 1)
	if err != nil {
		return errors.Wrap(Error(err), "error stopping container")
	}
	del := func() error {
		c.client.KillContainer(dc.KillContainerOptions{
			ID:     c.ID,
			Signal: dc.SIGKILL,
		})
		return c.client.RemoveContainer(dc.RemoveContainerOptions{
			ID:            c.ID,
			RemoveVolumes: true,
			Force:         true,
		})
	}
	defer del()
	if err := del(); err != nil {
		return errors.Wrap(Error(err), "error deleting container")
	}
	return nil
}
