package docker

import (
	"io"

	distreference "github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/rai-project/model"
)

func (c *Client) ImagePush(name string, pubOpts model.Publish, dockerReader io.Reader) (io.ReadCloser, error) {
	distributionRef, err := distreference.ParseNamed(name)
	if err != nil {
		return nil, err
	}

	if _, isCanonical := distributionRef.(distreference.Canonical); isCanonical {
		return nil, errors.New("cannot push a digest reference")
	}

	authOk, err := c.Client.RegistryLogin(c.options.context, types.AuthConfig{
		Username: pubOpts.Credentials.Username,
		Password: pubOpts.Credentials.Password,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to login registry using username = %s", pubOpts.Credentials.Username)
	}

	if authOk.Status == "" {
		return nil, errors.Wrapf(err, "unable to login registry because of invalid status code")
	}

	auth := &types.AuthConfig{
		Username:      pubOpts.Credentials.Username,
		Password:      pubOpts.Credentials.Password,
		IdentityToken: authOk.IdentityToken,
	}

	_ = auth
	// registry.NewSession()
	// cred := registry.NewStaticCredentialStore(auth)
	// cred.Basic(*url.URL)
	// resolved := registry.ResolveAuthConfig(authConfigs, index)
	// c.Client.ImagePush(c.options.context, name, types.ImagePushOptions{
	//   RegistryAuth: auth.IdentityToken,
	// })
	return nil, nil
}
