package docker

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/rai-project/config"
	"github.com/rai-project/docker/cuda"
	nvidiasmi "github.com/rai-project/nvidia-smi"
	uuid "github.com/rai-project/uuid"
	"github.com/spf13/cast"
)

type ContainerOptions struct {
	name            string
	runtime         string
	visibleGPUs     map[string]int
	containerConfig *container.Config
	hostConfig      *container.HostConfig
	networkConfig   *network.NetworkingConfig
	parentCtx       context.Context
	context         context.Context
	cancelFunc      context.CancelFunc
}

type ContainerOption func(*ContainerOptions)

var (
	DefaultContainerEnv = map[string]string{
		"CI":              "rai",
		"RAI":             "true",
		"RAI_ARCH":        filepath.Join(runtime.GOOS, runtime.GOARCH),
		"RAI_USER":        "root",
		"RAI_SOURCE_DIR":  "/src",
		"RAI_DATA_DIR":    "/data",
		"RAI_BUILD_DIR":   "/build",
		"DATA_DIR":        "/dir",
		"SOURCE_DIR":      "/src",
		"BUILD_DIR":       "/build",
		"TERM":            "xterm",
		"PATH":            "/usr/local/cuda/bin:/usr/local/nvidia/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"LD_LIBRARY_PATH": "/usr/local/cuda/lib64:/usr/local/nvidia/lib64",
		"SHELL":           "/bin/bash",
		"SHELLOPTS":       "braceexpand:emacs:hashall:histexpand:history:interactive-comments:monitor",
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
		//Shell: []string{
		//	"/bin/bash",
		//},
		Entrypoint: []string{
			"/bin/sh",
		},
		User:            Config.Username,
		AttachStdin:     c.options.stdin != nil,
		AttachStdout:    true,
		AttachStderr:    true,
		OpenStdin:       c.options.stdin != nil,
		StdinOnce:       false,
		Tty:             true,
		NetworkDisabled: true,
		WorkingDir:      "/build",
		StopSignal:      "SIGKILL",
		Volumes:         map[string]struct{}{},
	}
	hostConfig := &container.HostConfig{
		Privileged:      true,
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
		parentCtx:       c.options.context,
		context:         ctx,
		cancelFunc:      cancelFunc,
	}
}

func GPUCount(cnt int) ContainerOption {
	if GPUDeviceUsageState == nil {
		st, err := NewGPUUsageState()
		if err != nil {
			panic(err)
		}
		GPUDeviceUsageState = st
	}
	return func(o *ContainerOptions) {
		if o.containerConfig.Env == nil {
			o.containerConfig.Env = []string{}
		}
		devices := make([]string, cnt)
		o.visibleGPUs = make(map[string]int)
		for ii := 0; ii < cnt; ii++ {
			key, val, ok := GPUDeviceUsageState.RemoveOldest()
			if !ok {
				panic(errors.New("unable to remove oldest value from cache"))
			}
			o.visibleGPUs[cast.ToString(key)] = cast.ToInt(val)
			devices[ii] = strconv.Itoa(ii)
		}
		o.containerConfig.Env = append(
			o.containerConfig.Env,
			"CUDA_VISIBLE_DEVICES="+strings.Join(devices, ","),
		)
	}
}

func Runtime(s string) ContainerOption {
	return func(o *ContainerOptions) {
		o.hostConfig.Runtime = s
	}
}

func Tty(b bool) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Tty = b
	}
}

func OpenStdin(b bool) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.OpenStdin = b
	}
}

func Image(s string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Image = s
	}
}

func AddEnv(k, v string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Env = append(o.containerConfig.Env, k+"="+v)
	}
}

func User(u string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.User = u
	}
}

func Shell(s []string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Shell = s
	}
}

func Entrypoint(s []string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Entrypoint = s
	}
}

func Cmd(s []string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Cmd = s
	}
}

func Timelimit(t time.Duration) ContainerOption {
	return func(o *ContainerOptions) {
		o.cancelFunc()
		ctx, cancelFunc := context.WithTimeout(o.parentCtx, t)
		o.context = ctx
		o.cancelFunc = cancelFunc
	}
}

func ContainerConfig(h container.Config) ContainerOption {
	return func(o *ContainerOptions) {
		*o.containerConfig = h
	}
}

func HostConfig(h container.HostConfig) ContainerOption {
	return func(o *ContainerOptions) {
		*o.hostConfig = h
	}
}

func NetworkConfig(h network.NetworkingConfig) ContainerOption {
	return func(o *ContainerOptions) {
		*o.networkConfig = h
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

func AddVolume(dir string) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.Volumes[dir] = struct{}{}
	}
}

func Memory(n int64) ContainerOption {
	return func(o *ContainerOptions) {
		o.hostConfig.Resources.Memory = n
	}
}

func NetworkDisabled(b bool) ContainerOption {
	return func(o *ContainerOptions) {
		o.containerConfig.NetworkDisabled = b
	}
}

func CUDADevice(n int) ContainerOption {
	dev := fmt.Sprintf("/dev/nvidia%d", n)
	return Devices([]container.DeviceMapping{
		container.DeviceMapping{
			PathInContainer:   cuda.DeviceCtl,
			PathOnHost:        cuda.DeviceCtl,
			CgroupPermissions: "rwm",
		},
		container.DeviceMapping{
			PathInContainer:   cuda.DeviceUVM,
			PathOnHost:        cuda.DeviceUVM,
			CgroupPermissions: "rwm",
		},
		container.DeviceMapping{
			PathInContainer:   cuda.DeviceUVMTools,
			PathOnHost:        cuda.DeviceUVMTools,
			CgroupPermissions: "rwm",
		},
		container.DeviceMapping{
			PathInContainer:   dev,
			PathOnHost:        dev,
			CgroupPermissions: "rwm",
		},
	})
}

func ReadonlyRootfs(b bool) ContainerOption {
	return func(o *ContainerOptions) {
		o.hostConfig.ReadonlyRootfs = b
	}
}

func Device(d container.DeviceMapping) ContainerOption {
	return Devices([]container.DeviceMapping{d})
}

func Devices(ds []container.DeviceMapping) ContainerOption {
	return func(o *ContainerOptions) {
		add := func(d container.DeviceMapping) {
			for _, e := range o.hostConfig.Resources.Devices {
				if e.PathInContainer == d.PathInContainer &&
					e.PathOnHost == d.PathOnHost &&
					e.CgroupPermissions == d.CgroupPermissions {
					return
				}
			}
			o.hostConfig.Resources.Devices = append(
				o.hostConfig.Resources.Devices,
				d,
			)
		}
		for _, d := range ds {
			add(d)
		}
	}
}

func NvidiaVolume(version string) ContainerOption {
	return func(o *ContainerOptions) {
		if version == "" && nvidiasmi.HasGPU {
			version = nvidiasmi.Info.DriverVersion
		}
		//o.hostConfig.VolumeDriver = "rai-cuda"
		name := "rai-cuda"
		versionedName := name + "_" + version
		_ = versionedName
		o.hostConfig.Mounts = append(
			o.hostConfig.Mounts,
			mount.Mount{
				Type:     mount.TypeVolume,
				Source:   versionedName,
				Target:   "/usr/local/nvidia",
				ReadOnly: true,
				VolumeOptions: &mount.VolumeOptions{
					Labels: map[string]string{
						"name": versionedName,
					},
					DriverConfig: &mount.Driver{
						Name: name,
					},
				},
			},
		)
	}
}
