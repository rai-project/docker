package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/imdario/mergo"
	"github.com/rai-project/config"
	uuid "github.com/satori/go.uuid"
)

type ContainerOptions struct {
	name            string
	containerConfig *container.Config
	hostConfig      *container.HostConfig
	networkConfig   *network.NetworkingConfig
	context         context.Context
	cancelFunc      context.CancelFunc
}

type ContainerOption func(*ContainerOptions)

var (
	DefaultContainerEnv = map[string]string{
		"CI":             "rai",
		"RAI":            "true",
		"RAI_ARCH":       filepath.Join(runtime.GOOS, runtime.GOARCH),
		"RAI_USER":       "root",
		"RAI_SOURCE_DIR": "/src",
		"RAI_DATA_DIR":   "/data",
		"RAI_BUILD_DIR":  "/build",
		"DATA_DIR":       "/dir",
		"SOURCE_DIR":     "/src",
		"BUILD_DIR":      "/build",
		"TERM":           "xterm",
		"PATH":           "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}
)

func getEnv() []string {
	envMap := map[string]string{}
	mergo.MergeWithOverwrite(&envMap, DefaultContainerEnv)
	mergo.MergeWithOverwrite(&envMap, Config.Env)

	env := []string{}
	for k, v := range envMap {
		env = append(env, k+"="+v)
	}
	return env
}

func NewContainerOptions(c *Client) *ContainerOptions {
	containerConfig := &container.Config{
		Hostname: fmt.Sprintf("%s-run-%s", config.App.Name, uuid.NewV4()),
		Env:      getEnv(),
		Image:    Config.Image,
		Shell: []string{
			"/bin/bash",
		},
		User:            Config.Username,
		AttachStdin:     false,
		AttachStdout:    true,
		AttachStderr:    true,
		OpenStdin:       true,
		StdinOnce:       true,
		Tty:             true,
		NetworkDisabled: true,
	}
	hostConfig := &container.HostConfig{
		Privileged:      false,
		AutoRemove:      false,
		PublishAllPorts: false,
		ReadonlyRootfs:  false,
		Resources: container.Resources{
			Memory:     Config.MemoryLimit,
			MemorySwap: -1,
			Devices:    []container.DeviceMapping{},
		},
		Binds: []string{},
		CapDrop: []string{ // see http://rhelblog.redhat.com/2016/10/17/secure-your-containers-with-this-one-weird-trick/
			"chown",
			"dac_override",
			"fowner",
			"fsetid",
			"setgid",
			"setuid",
			"setpcap",
			"net_bind_service",
			"net_raw",
			"sys_chroot",
			"mknod",
			"audit_write",
			"setfcap",
		},
	}
	networkConfig := &network.NetworkingConfig{}
	ctx, cancelFunc := context.WithTimeout(c.options.context, Config.TimeLimit)
	return &ContainerOptions{
		containerConfig: containerConfig,
		hostConfig:      hostConfig,
		networkConfig:   networkConfig,
		context:         ctx,
		cancelFunc:      cancelFunc,
	}
}

func Hostname(h string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Hostname = h
	}
}

func WorkingDirectory(dir string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.WorkingDir = dir
	}
}

func CUDAVolume() ContainerOption {
	return func(o *ContainerOptions) {
		//...
	}
}
