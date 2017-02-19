package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContainer(t *testing.T) {

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.PullImage(Config.Repository, Config.Tag)
	assert.NoError(t, err)

	cont, err := NewContainer(client)
	assert.NoError(t, err)
	assert.NotNil(t, cont)

}

func TestStartContainer(t *testing.T) {

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	cont, err := NewContainer(client)
	assert.NoError(t, err)
	assert.NotNil(t, cont)

	defer func() {
		err := cont.Stop()
		assert.NoError(t, err)
	}()

	err = cont.Start()
	assert.NoError(t, err)
}
