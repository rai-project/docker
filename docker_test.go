package docker

import (
	"os"
	"testing"

	"github.com/rai-project/config"
)

func TestMain(m *testing.M) {
	config.Init()
	os.Exit(m.Run())
}
