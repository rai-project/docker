package docker

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/rai-project/model"
)

func (c *Client) ImagePush(name0 string, pushOpts model.Push) (io.ReadCloser, error) {
	name, err := parseImageName(name0)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse the image name %v", name0)
	}

	log.WithField("image_name", name).Debug("publishing to docker repository")

	username := pushOpts.Credentials.Username
	password := pushOpts.Credentials.Password
	email := ""
	if strings.Contains(username, "@") {
		email = username
		username = ""
	}
	authOk, err := c.Client.RegistryLogin(c.options.context, types.AuthConfig{
		Username: username,
		Password: password,
		Email:    email,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to login registry using username = %s", username)
	}

	if authOk.Status == "" {
		return nil, errors.Wrapf(err, "unable to login registry because of invalid status code")
	}

	auth := types.AuthConfig{
		Username:      username,
		Password:      password,
		Email:         email,
		IdentityToken: authOk.IdentityToken,
	}

	encodedJSON, err := json.Marshal(auth)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal auth to json")
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	return c.Client.ImagePush(c.options.context, name, types.ImagePushOptions{RegistryAuth: authStr})
}
