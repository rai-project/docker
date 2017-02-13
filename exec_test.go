package docker

import (
	"bytes"
	"testing"

	"bitbucket.org/hwuligans/rai/pkg/config"

	"github.com/stretchr/testify/assert"
)

func TestExecution(t *testing.T) {
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
