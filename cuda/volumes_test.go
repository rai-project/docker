package cuda

import (
	"testing"

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
