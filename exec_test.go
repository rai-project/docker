package docker

import (
	"bytes"
	"os"
	"testing"

	"github.com/k0kubun/pp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExecTestSuite struct {
	suite.Suite
	client    *Client
	container *Container
}

func NewExecTestSuite(t *testing.T) (*ExecTestSuite, error) {
	client, err := NewClient(
		Stdout(os.Stdout),
		Stderr(os.Stderr),
	)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}

	cont, err := NewContainer(client)
	assert.NoError(t, err)
	if err != nil {
		return nil, err
	}
	return &ExecTestSuite{
		client:    client,
		container: cont,
	}, nil
}

func (suite *ExecTestSuite) TestRun() {
	t := suite.T()
	cont := suite.container

	defer func() {
		err := cont.Stop()
		assert.NoError(t, err)
	}()

	err := cont.Start()
	assert.NoError(t, err)

	// exec, err := NewExecution(cont, "/bin/sh", "-c", "ls", "-l", "/")

	exec, err := NewExecution(cont, "/bin/sh", "-c", "echo", "cat")
	assert.NoError(t, err)
	assert.NotNil(t, exec)

	var stdout, stderr bytes.Buffer
	exec.Stderr = &stderr
	exec.Stdout = &stdout

	err = exec.Run()
	assert.NoError(t, err, "execution should not return an error")
	assert.Empty(t, stderr.Bytes())
	assert.NotEmpty(t, stdout.Bytes())

	pp.Println(stderr.String())
	pp.Println(stdout.String())

	// cont.Info()
}

/*
func TestExecutionOutput(t *testing.T) {

	config.Init()

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

	exec, err := NewExecutionFromString(cont, "ls -l")
	assert.NoError(t, err)
	assert.NotNil(t, exec)

	out, err := exec.Output()
	assert.NoError(t, err, "execution should not return an error")
	assert.NotEmpty(t, out, "the output cannot be nil")

}

func TestExecutionOutput2(t *testing.T) {

	config.Init()

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

	run := func(cmd string) {

		exec, err := NewExecutionFromString(cont, cmd)
		assert.NoError(t, err)
		assert.NotNil(t, exec)

		out, err := exec.Output()
		assert.NoError(t, err, "execution should not return an error")
		assert.NotEmpty(t, out, "the output cannot be nil")

	}
	run("ls -l")
	run("ls -l /")

}
*/

func TestExec(t *testing.T) {
	c, err := NewExecTestSuite(t)
	if !assert.NoError(t, err, "Failed to create docker client") {
		return
	}
	suite.Run(t, c)
}
