package docker

import (
	"context"
	"io"

	"github.com/rai-project/uuid"
)

type BuildOptions struct {
	id             string
	cache          bool
	dockerFilePath string
	tags           []string
	labels         map[string]string
	args           map[string]string
	archiveReader  io.Reader
	quiet          bool
	context        context.Context
}

type BuildOption func(*BuildOptions)

func BuildId(id string) BuildOption {
	return func(opts *BuildOptions) {
		opts.id = id
	}
}

func BuildCache(cache bool) BuildOption {
	return func(opts *BuildOptions) {
		opts.cache = cache
	}
}

func BuildLabels(labels map[string]string) BuildOption {
	return func(opts *BuildOptions) {
		for k, v := range labels {
			opts.labels[k] = v
		}
	}
}

func BuildTags(tags []string) BuildOption {
	return func(opts *BuildOptions) {
		opts.tags = append(opts.tags, tags...)
	}
}

func BuildArguments(args map[string]string) BuildOption {
	return func(opts *BuildOptions) {
		for k, v := range args {
			opts.args[k] = v
		}
	}
}

func BuildArchiveReader(reader io.Reader) BuildOption {
	return func(opts *BuildOptions) {
		opts.archiveReader = reader
	}
}

func BuildDockerFilePath(path string) BuildOption {
	return func(opts *BuildOptions) {
		opts.dockerFilePath = path
	}
}

func BuildQuiet(quiet bool) BuildOption {
	return func(opts *BuildOptions) {
		opts.quiet = quiet
	}
}

func BuildContext(ctx context.Context) BuildOption {
	return func(opts *BuildOptions) {
		opts.context = ctx
	}
}

func NewBuildOptions(opts ...BuildOption) *BuildOptions {
	res := &BuildOptions{
		id:             uuid.NewV4(),
		cache:          true,
		dockerFilePath: "",
		tags:           []string{},
		labels:         map[string]string{},
		args:           map[string]string{},
		archiveReader:  nil,
		quiet:          false,
		context:        nil,
	}
	for _, o := range opts {
		o(res)
	}
	return res
}
