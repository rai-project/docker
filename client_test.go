package docker

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	client *Client
}

func NewClientTestSuite(t *testing.T) (*ClientTestSuite, error) {
	client, err := NewClient(
		Stdout(os.Stdout),
		Stderr(os.Stderr),
	)
	if err != nil {
		return nil, err
	}
	return &ClientTestSuite{
		client: client,
	}, nil
}

func (suite *ClientTestSuite) TestClientCreation() {
	client, err := NewClient()
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), client)

	info, err := client.Info(context.Background())
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), info)
}

func (suite *ClientTestSuite) TestRemoveImage() {
	client := suite.client
	err := client.RemoveImage("ubuntu:latest")
	assert.NoError(suite.T(), err)

	err = client.RemoveImage("rethinkdb")
	assert.NoError(suite.T(), err)
}

func (suite *ClientTestSuite) TestPullImage() {
	client := suite.client
	err := client.PullImage("ubuntu:latest")
	assert.NoError(suite.T(), err)

	err = client.PullImage("rethinkdb")
	assert.NoError(suite.T(), err)
}

func (suite *ClientTestSuite) TestListImages() {
	client := suite.client

	err := client.PullImage("ubuntu:latest")
	assert.NoError(suite.T(), err)

	imgs, err := client.ListImages("ubuntu")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), imgs)

	imgs, err = client.ListImages("ubuntu:latest")
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), imgs)
}

func TestClient(t *testing.T) {
	c, err := NewClientTestSuite(t)
	if !assert.NoError(t, err, "Failed to create docker client") {
		return
	}
	suite.Run(t, c)
}
