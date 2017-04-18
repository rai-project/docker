package docker

import (
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/command/image/build"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	cache "github.com/patrickmn/go-cache"
)

var (
	imageBuildCache *cache.Cache
)

func (c *Client) ImageBuild(name string, dockerReader io.Reader) error {

	// Setup an upload progress bar
	out := c.options.stdout
	progressOutput := streamformatter.NewStreamFormatter().NewProgressOutput(out, true)
	if !out.IsTerminal() {
		progressOutput = &lastProgressOutput{output: progressOutput}
	}

	buildCtx, relDockerfile, err = build.GetContextFromReader(config, "Dockerfile")

	var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	buildOptions := types.ImageBuildOptions{
		Tags: []string{name},
	}

	response, err := c.Client.ImageBuild(c.options.context, body, buildOptions)
	if err != nil {
		fmt.Fprintf(c.options.stderr, "%s", progBuff)
		return err
	}
	defer response.Body.Close()

	return nil
}

func (c *Client) ImageBuildCached(name string, dockerReader io.Reader) error {
	if imageBuildCache == nil {
		return c.ImageBuild(name, dockerReader)
	}

	if build, found := imageBuildCache.Get(name); found {
		return nil
	}

	err := c.ImageBuild(name, dockerReader)
	if err != nil {
		return err
	}
	imageBuildCache.Set(name, time.Now(), cache.DefaultExpiration)

	return nil

}

func init() {
	imageBuildCache = cache.New(5*time.Hour, 24*time.Hour)
}
