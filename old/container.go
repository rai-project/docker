package docker

import (
	"fmt"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/rai-project/config"
	"github.com/rai-project/uuid"
)

type Container struct {
	ImageName string
	Tag       string
	Context   context.Context
	client    *Client
	cancel    context.CancelFunc
	options   ContainerOptions
	*dc.Container
}

func NewContainer(client *Client) (*Container, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Config.TimeLimit)
	env := baseEnvsStringList()
	name := fmt.Sprintf("%s-run-%s", config.App.Name, uuid.NewV4())
	img := fmt.Sprintf("%s:%s", Config.Repository, Config.Tag)
	opts := dc.CreateContainerOptions{
		Name:       name,
		HostConfig: hostConfig,
		Config: &dc.Config{
			Image:           img,
			Hostname:        name,
			User:            Config.User,
			Env:             env,
			Memory:          Config.MemoryLimit,
			WorkingDir:      "/build",
			AttachStdin:     false,
			AttachStdout:    true,
			AttachStderr:    true,
			OpenStdin:       true,
			StdinOnce:       true,
			Tty:             true,
			NetworkDisabled: true,
			// VolumeDriver:    "nvidia-docker",
			// Mounts: []dc.Mount{
			// 	dc.Mount{
			// 		Name:        "nvidia_driver_367.57",
			// 		Source:      "/var/lib/nvidia-docker/volumes/nvidia_driver/367.57",
			// 		Destination: "/usr/local/nvidia",
			// 		Driver:      "nvidia-docker",
			// 		Mode:        "ro",
			// 		RW:          false,
			// 	},
			// },
			// Cmd: []string{
			// 	"sleep",
			// 	"4h",
			// },
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
		log.WithError(err).
			WithField("image", img).
			Error("Failed to create docker container.")
		return nil, errors.Wrap(err, "Failed to create docker container.")
	}
	res := &Container{
		ImageName: Config.Repository,
		Tag:       Config.Tag,
		Container: cont,
		client:    client,
		cancel:    cancel,
		Context:   ctx,
	}
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			res.Stop()
		}
	}()
	return res, nil
}

func (c *Container) Start() error {
	return c.client.StartContainer(c.ID, hostConfig)
}

func (c *Container) Stop() error {
	defer c.cancel()
	log.WithField("container", c.ID).Debug("stopping container")
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
