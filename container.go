package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/k0kubun/pp"
)

type Container struct {
	ID        string
	isStarted bool
	client    *Client
	options   ContainerOptions
}

func NewContainer(client *Client, paramOpts ...ContainerOption) (*Container, error) {
	options := NewContainerOptions(client)
	for _, o := range paramOpts {
		o(options)
	}
	if !client.HasImage(options.containerConfig.Image) {
		err := client.PullImage(options.containerConfig.Image)
		if err != nil {
			return nil, err
		}
	}
	c, err := client.ContainerCreate(
		options.context,
		options.containerConfig,
		options.hostConfig,
		options.networkConfig,
		options.name,
	)
	if err != nil {
		return nil, err
	}
	container := &Container{
		ID:      c.ID,
		client:  client,
		options: *options,
	}
	go func() {
		ctx := container.options.context
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			container.Stop()
		}
	}()
	return container, nil
}

func (c *Container) Start() error {
	client := c.client
	err := client.ContainerStart(
		c.options.context,
		c.ID,
		types.ContainerStartOptions{},
	)
	if err != nil {
		return err
	}
	c.isStarted = true
	return nil
}

func (c *Container) Info() (types.ContainerJSON, error) {
	client := c.client
	info, err := client.ContainerInspect(
		c.options.context,
		c.ID,
	)
	if err != nil {
		return types.ContainerJSON{}, err
	}
	pp.Println(info)
	return info, nil
}

func (c *Container) Close() error {
	return c.Stop()
}

func (c *Container) Stop() error {
	c.options.cancelFunc()
	if !c.isStarted {
		return nil
	}
	defer func() {
		c.isStarted = false
	}()
	c.kill()
	return c.remove()
}

func (c *Container) killWithSignal(sig string) error {
	client := c.client
	err := client.ContainerKill(
		c.options.context,
		c.ID,
		sig,
	)
	return err
}

func (c *Container) kill() error {
	return c.killWithSignal("SIGKILL")
}

func (c *Container) remove() error {
	client := c.client
	err := client.ContainerRemove(
		c.options.context,
		c.ID,
		types.ContainerRemoveOptions{
			// RemoveVolumes: true,
			// RemoveLinks:   true,
			Force: true,
		},
	)
	return err
}

func (c *Container) Options() ContainerOptions {
	return c.options
}
