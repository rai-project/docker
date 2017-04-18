package cuda

import (
	"os"
	"testing"

	"github.com/rai-project/config"
	nvidiasmi "github.com/rai-project/nvidia-smi"
	"github.com/stretchr/testify/assert"
)

func TestGetVolume(t *testing.T) {
	version := nvidiasmi.Info.DriverVersion
	volume, ver, err := getVolume("rai-cuda_" + version)
	assert.NoError(t, err)
	assert.Equal(t, version, ver)
	assert.NotNil(t, volume)
}

func TestMain(m *testing.M) {
	config.Init(
		config.VerboseMode(true),
		config.DebugMode(true),
	)
	os.Exit(m.Run())

}
