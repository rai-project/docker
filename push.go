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

func (c *Client) ImagePush(name0 string, pubOpts model.Publish, dockerReader io.Reader) (io.ReadCloser, error) {
	name, err := parseImageName(name0)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse the image name %v", name0)
	}

	log.WithField("image_name", name).Debug("publishing to docker repository")

	username := pubOpts.Credentials.Username
	password := pubOpts.Credentials.Password
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
		return nil, errors.Wrapf(err, "unable to login registry using username = %s", pubOpts.Credentials.Username)
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

	// pp.Println(authOk)

	// indexServer := pubOpts.Registry
	// if indexServer == "" {
	// 	indexServer = registry.IndexServer
	// }

	// indexInfo, err := registry.ParseSearchIndexInfo(indexServer)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "unable to parse the search index info for %s", indexServer)
	// }

	// pp.Println(indexInfo)

	// resolved := registry.ResolveAuthConfig(
	// 	map[string]types.AuthConfig{
	// 		indexServer: auth,
	// 	},
	// 	&registrytypes.IndexInfo{
	// 		Name:     indexServer,
	// 		Official: true,
	// 	},
	// )

	// registry.NewSession()
	// cred := registry.NewStaticCredentialStore(auth)
	// cred.Basic(*url.URL)
	// resolved := registry.ResolveAuthConfig(authConfigs, index)
	// c.Client.ImagePush(c.options.context, name, types.ImagePushOptions{
	//   RegistryAuth: auth.IdentityToken,
	// })
	return nil, nil
}
