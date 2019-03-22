package docker

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
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

func (c *Client) ImageBuild(iopts ...BuildOption) error {

	opts := NewBuildOptions(iopts...)

	if opts.context == nil {
		opts.context = c.options.context
	}

	// Setup an upload progress bar
	stdout := c.options.stdout
	if stdout == nil || opts.quiet {
		stdout = NewOutStream(ioutil.Discard)
	}
	stderr := c.options.stderr
	if stderr == nil || opts.quiet {
		stderr = NewOutStream(ioutil.Discard)
	}

	progressOutput := streamformatter.NewProgressOutput(streamformatter.NewStdoutWriter(stdout))

	buildCtx, _, err := getContextFromReader(ioutil.NopCloser(opts.archiveReader), opts.dockerFilePath)
	if err != nil {
		return err
	}

	var body io.Reader = progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	buildArgs := map[string]*string{}
	for k, v := range opts.args {
		val := v
		buildArgs[k] = &val
	}

	buildOptions := types.ImageBuildOptions{
		BuildID:        opts.id,
		Dockerfile:     opts.dockerFilePath,
		Tags:           opts.tags,
		Labels:         opts.labels,
		BuildArgs:      buildArgs,
		SuppressOutput: opts.quiet,
		NoCache:        !opts.cache,
	}

	response, err := c.Client.ImageBuild(opts.context, body, buildOptions)
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

func (c *Client) ImageBuildCached(iopts ...BuildOption) error {

	opts := NewBuildOptions(iopts...)

	if imageBuildCache == nil || len(opts.tags) == 0 {
		return c.ImageBuild(iopts...)
	}

	name := strings.Join(opts.tags, ";")

	if _, found := imageBuildCache.Get(name); found {
		return nil
	}

	err := c.ImageBuild(iopts...)
	if err != nil {
		return err
	}

	imageBuildCache.Set(name, time.Now(), cache.DefaultExpiration)

	return nil

}

func init() {
	imageBuildCache = cache.New(5*time.Hour, 24*time.Hour)
}
