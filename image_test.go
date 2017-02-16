package docker

import (
	"testing"

	"github.com/rai-project/config"

	"github.com/stretchr/testify/assert"
)

func TestDockerListImages(t *testing.T) {
	config.Init()

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	_, err = client.ListImages("image", "tag")
	assert.NoError(t, err)
}
