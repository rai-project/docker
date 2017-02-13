package docker

import (
	"testing"

	"bitbucket.org/hwuligans/rai/pkg/config"

	"github.com/stretchr/testify/assert"
)

func XXXTestDockerExecution(t *testing.T) {
	config.Init()

	client, err := NewClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// exe, err := NewExecution()
	// assert.NoError(t, err)
	// assert.NotNil(t, exe)

	// defer exe.Cancel()
	// run := func(shellcmd string) {
	// 	args, err := shellwords.Parse(shellcmd)
	// 	assert.NoError(t, err)
	// 	cmd := client.Command(exe, args[0], args[1:]...)
	// 	cmd.Stdout = os.Stdout
	// 	err = cmd.Run()
	// 	assert.NoError(t, err)
	// }
	// run("ls -l")
	// run("ls -l /")
}
