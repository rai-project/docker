package docker

import (
	"testing"

	"bitbucket.org/hwuligans/rai/pkg/config"

	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
)

func XXXTestContainer(t *testing.T) {
	config.Init()

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	pp.Println("creating new container")
	cont, err := NewContainer(client)
	assert.NoError(t, err)
	assert.NotNil(t, cont)
	pp.Println("created new container")

	defer func() {
		err := cont.Stop()
		assert.NoError(t, err)
	}()

	pp.Println("starting container")
	err = cont.Start()
	assert.NoError(t, err)
	pp.Println("started container")

}
