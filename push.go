package docker

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pkg/errors"
	"github.com/rai-project/config"
	"github.com/rai-project/model"
	"github.com/rai-project/utils"
)

func (c *Client) ImagePush(name0 string, pushOpts model.Push) error {
	decrypt := func(s string) string {
		if strings.HasPrefix(s, utils.CryptoHeader) && config.App.Secret != "" {
			if val, err := utils.DecryptStringBase64(config.App.Secret, s); err == nil {
				return val
			}
		}
		if r, err := base64.StdEncoding.DecodeString(s); err == nil {
			return string(r)
		}
		return s
	}

	name, err := parseImageName(name0)
	if err != nil {
		return errors.Wrapf(err, "unable to parse the image name %v", name0)
	}

	log.WithField("image_name", name).Debug("publishing to docker repository")

	username := decrypt(pushOpts.Credentials.Username)
	password := decrypt(pushOpts.Credentials.Password)
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
		return errors.Wrapf(err, "unable to login registry using username = %s", username)
	}

	if authOk.Status == "" {
		return errors.Wrapf(err, "unable to login registry because of invalid status code")
	}

	auth := types.AuthConfig{
		Username:      username,
		Password:      password,
		Email:         email,
		IdentityToken: authOk.IdentityToken,
	}

	encodedJSON, err := json.Marshal(auth)
	if err != nil {
		return errors.Wrap(err, "unable to marshal auth to json")
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	reader, err := c.Client.ImagePush(c.options.context, name, types.ImagePushOptions{RegistryAuth: authStr})
	if err != nil {
		return err
	}

	defer reader.Close()

	return jsonmessage.DisplayJSONMessagesToStream(
		reader,
		c.options.stdout,
		nil,
	)
}
