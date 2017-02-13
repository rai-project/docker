package docker

import (
	"testing"

	"bitbucket.org/hwuligans/rai/pkg/config"

	"github.com/stretchr/testify/assert"
)

func TestDockerListImages(t *testing.T) {
	config.Init()

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.ListImages()
	assert.NoError(t, err)
}
