package docker

import (
	"testing"

	"github.com/rai-project/config"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	config.Init(
		config.VerboseMode(true),
		config.DebugMode(true),
	)

	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("github.com/patrickmn/go-cache.(*janitor).Run"),
		goleak.IgnoreTopFunction("github.com/rai-project/docker/vendor/github.com/patrickmn/go-cache.(*janitor).Run"),
	)

}
