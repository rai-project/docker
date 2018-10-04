package docker

import (
	"bytes"
	"os"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ExecTestSuite struct {
	suite.Suite
	client *Client
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

	return &ExecTestSuite{
		client: client,
	}, nil
}

func (suite *ExecTestSuite) TestRun() {
	t := suite.T()
	client := suite.client

	cont, err := NewContainer(client)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	defer func() {
		err := cont.Stop()
		assert.NoError(t, err)
	}()

	err = cont.Start()
	assert.NoError(t, err)

	exec, err := NewExecution(cont, "ls", "/")

	assert.NoError(t, err)
	assert.NotNil(t, exec)

	var stdout, stderr bytes.Buffer
	exec.Stderr = &stderr
	exec.Stdout = &stdout

	err = exec.Run()
	assert.NoError(t, err, "execution should not return an error")

	assert.Empty(t, stderr.Bytes())
	assert.NotEmpty(t, stdout.Bytes())

	out := stdout.String()
	dirs := strings.FieldsFunc(out, unicode.IsSpace)
	sort.Strings(dirs)
	expected := []string{
		"bin",
		"boot",
		"build",
		"dev",
		"etc",
		"home",
		"lib",
		"lib64",
		"media",
		"mnt",
		"opt",
		"proc",
		"root",
		"run",
		"sbin",
		"srv",
		"sys",
		"tmp",
		"usr",
		"var",
	}
	assert.Equal(t, dirs, expected)

}

func (suite *ExecTestSuite) TestExecutionOutput() {
	t := suite.T()
	client := suite.client

	cont, err := NewContainer(client)
	assert.NoError(t, err)
	if err != nil {
		return
	}

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

/*
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
	defer c.client.Close()
	suite.Run(t, c)
}
