package docker

import (
	"os"
	"testing"

	"github.com/rai-project/config"
)

func TestMain(m *testing.M) {
	os.Setenv("DEBUG", "TRUE")
	os.Setenv("VERBOSE", "TRUE")
	config.Init()
	os.Exit(m.Run())
}
