package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/rai-project/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var testPushModel model.Publish

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

	readCloser, err := client.ImagePush(testPushModel.ImageName, testPushModel, client.options.stdin)
	if !assert.NoError(t, err, "Failed to push image") {
		return
	}
	if !assert.NotNil(t, readCloser, "Returned valid readCloser") {
		return
	}

	defer readCloser.Close()

	bts, err := ioutil.ReadAll(readCloser)
	if !assert.NoError(t, err, "Failed to read push output") {
		return
	}
	assert.NotEmpty(t, bts, "empty push output")

	fmt.Println(string(bts))
}

func TestPush(t *testing.T) {
	c, err := NewPushTestSuite(t)
	if !assert.NoError(t, err, "Failed to create docker Push test suite") {
		return
	}
	suite.Run(t, c)
}
