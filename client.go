package docker

import (
	"net/http"

	"github.com/docker/distribution/reference"
	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/filters"
	dc "github.com/moby/moby/client"
	"github.com/moby/moby/pkg/jsonmessage"
	"github.com/pkg/errors"
)

type Client struct {
	*dc.Client
	options ClientOptions
}

func NewClient(paramOpts ...ClientOption) (*Client, error) {
	opts := NewClientOptions()
	for _, o := range paramOpts {
		o(opts)
	}

	var httpClient *http.Client
	if opts.tlsConfig != nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: opts.tlsConfig,
			},
		}
	}
	client, err := dc.NewClient(opts.host, opts.apiVersion, httpClient, nil)
	if err != nil {
		log.WithError(err).Error("Not able to create docker client")
		return nil, err
	}
	return &Client{
		Client:  client,
		options: *opts,
	}, nil
}

func parseImageName(refName string) (string, error) {
	ref, err := reference.Parse(refName)
	if err != nil {
		return "", err
	}
	if named, ok := ref.(reference.Named); ok {
		ref = reference.TagNameOnly(named)
	}
	return ref.String(), nil
}

func (c *Client) ListImages(refName string) ([]types.ImageSummary, error) {
	ref, err := parseImageName(refName)
	if err != nil {
		return nil, err
	}
	filter := filters.NewArgs()
	filter.Add("reference", ref)
	imgs, err := c.ImageList(
		c.options.context,
		types.ImageListOptions{
			All:     false,
			Filters: filter,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get list of images from docker")
	}
	return imgs, nil
}

func (c *Client) HasImage(refName string) bool {
	imgs, err := c.ListImages(refName)
	if err != nil {
		return false
	}
	return len(imgs) > 0
}

func (c *Client) PullImage(refName string) error {
	ref, err := reference.Parse(refName)
	if err != nil {
		return err
	}
	if tagged, ok := ref.(reference.Tagged); ok {
		if tagged.Tag() != "latest" && c.HasImage(refName) {
			c.options.stdout.Write([]byte("The docker image " + ref.String() + " was found on the host system\n."))
			return nil
		}
	}
	responseBody, err := c.ImagePull(
		c.options.context,
		ref.String(),
		types.ImagePullOptions{},
	)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesToStream(
		responseBody,
		c.options.stdout,
		nil,
	)
}

func (c *Client) RemoveImage(refName string) error {
	if !c.HasImage(refName) {
		return nil
	}
	ref, err := parseImageName(refName)
	if err != nil {
		return err
	}
	_, err = c.ImageRemove(
		c.options.context,
		ref,
		types.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		},
	)
	return err
}
