package docker

import dc "github.com/fsouza/go-dockerclient"

type Client struct {
	*dc.Client
}

func NewClient() (*Client, error) {
	client, err := dc.NewClientFromEnv()
	if err != nil {
		log.WithError(err).Error("Not able to create docker client")
		return nil, err
	}
	c := &Client{client}
	return c, nil
}
