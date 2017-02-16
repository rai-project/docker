package docker

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	sourcepath "github.com/GeertJohan/go-sourcepath"
	"github.com/rai-project/config"

	"github.com/stretchr/testify/assert"
)

var testYamlPath = "config_test.yml"
var testYaml = `docker:
    repository: "ubuntu"
    tag: "latest"
    user: ""
`

func TestMain(m *testing.M) {
	// flag.Parse()
	bytes := []byte(testYaml)
	err := ioutil.WriteFile(testYamlPath, bytes, 0777)

	if err != nil {
		panic("cannot create temp yaml config file needed for testing")
	}
	defer os.Remove(testYamlPath)

	config.ConfigFileName = filepath.Join(sourcepath.MustAbsoluteDir(), testYamlPath)
	config.Init()

	os.Exit(m.Run())
}

func TestExecution(t *testing.T) {

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	err = client.PullImage(Config.Repository, Config.Tag)
	assert.NoError(t, err)

	cont, err := NewContainer(client)
	assert.NoError(t, err)
	assert.NotNil(t, cont)

	defer func() {
		err := cont.Stop()
		assert.NoError(t, err)
	}()

	err = cont.Start()
	assert.NoError(t, err)

	exec, err := NewExecution(cont, "ls", "-l")
	assert.NoError(t, err)
	assert.NotNil(t, exec)

	var stdout, stderr bytes.Buffer
	exec.Stderr = &stderr
	exec.Stdout = &stdout

	err = exec.Run()
	assert.NoError(t, err, "execution should not return an error")
	assert.Empty(t, stderr.Bytes())
	assert.NotEmpty(t, stdout.Bytes())
}

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
