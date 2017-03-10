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
	volume, ver, err := getVolume("nvidia_driver_" + version)
	assert.NoError(t, err)
	assert.Equal(t, version, ver)
	assert.NotNil(t, volume)
}

func TestMain(m *testing.M) {
	os.Setenv("DEBUG", "TRUE")
	os.Setenv("VERBOSE", "TRUE")
	config.Init()
	os.Exit(m.Run())

}
