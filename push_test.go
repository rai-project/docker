package docker

import (
	"os"
	"testing"

	"github.com/rai-project/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var testPushModel model.Push

type PushTestSuite struct {
	suite.Suite
	client *Client
}

func NewPushTestSuite(t *testing.T) (*PushTestSuite, error) {
	client, err := NewClient(
		Stdout(os.Stdout),
		Stderr(os.Stderr),
	)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	return &PushTestSuite{
		client: client,
	}, nil
}

func (suite *PushTestSuite) TestAuthentication() {

	t := suite.T()
	client := suite.client

	err := client.ImagePush(testPushModel.ImageName, testPushModel)
	if !assert.NoError(t, err, "Failed to push image") {
		return
	}
}

func DISABLED_TestPush(t *testing.T) {
	c, err := NewPushTestSuite(t)
	if !assert.NoError(t, err, "Failed to create docker Push test suite") {
		return
	}
	suite.Run(t, c)
}
