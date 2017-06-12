package docker

import (
	"fmt"
	"io"
	"io/ioutil"
	"time"

	 "github.com/docker/docker/api/types"
	 "github.com/docker/docker/pkg/jsonmessage"
	 "github.com/docker/docker/pkg/progress"
	 "github.com/docker/docker/pkg/streamformatter"
	cache "github.com/patrickmn/go-cache"
)

var (
	imageBuildCache *cache.Cache
)

func (c *Client) ImageBuild(name string, dockerReader io.Reader) error {

	// Setup an upload progress bar
	stdout := c.options.stdout
	if stdout == nil {
		stdout = NewOutStream(ioutil.Discard)
	}
	stderr := c.options.stderr
	if stderr == nil {
		stderr = NewOutStream(ioutil.Discard)
	}

	progressOutput := streamformatter.NewProgressOutput(streamformatter.NewStdoutWriter(stdout))

	buildCtx, _, err := getContextFromReader(ioutil.NopCloser(dockerReader), "Dockerfile")
	if err != nil {
		return err
	}

	var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	buildOptions := types.ImageBuildOptions{
		Tags: []string{name},
	}

	response, err := c.Client.ImageBuild(c.options.context, body, buildOptions)
	if err != nil {
		fmt.Fprintf(stderr, "%v", err)
		return err
	}

	defer response.Body.Close()

	err = jsonmessage.DisplayJSONMessagesStream(response.Body, stdout, stdout.FD(), stdout.IsTerminal(), nil)
	if err != nil {
		fmt.Fprintf(stderr, "%v", err)
		return err
	}

	return nil
}

func (c *Client) ImageBuildCached(name string, dockerReader io.Reader) error {
	if imageBuildCache == nil {
		return c.ImageBuild(name, dockerReader)
	}

	if _, found := imageBuildCache.Get(name); found {
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
