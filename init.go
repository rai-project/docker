package docker

import (
	"github.com/Sirupsen/logrus"
	"github.com/Unknwon/com"
	"github.com/davecgh/go-spew/spew"
	dc "github.com/fsouza/go-dockerclient"

	"bitbucket.org/hwuligans/rai/pkg/config"
	logger "bitbucket.org/hwuligans/rai/pkg/logger"
	"bitbucket.org/hwuligans/rai/pkg/utils"
)

var (
	client *dc.Client

	hostConfig *dc.HostConfig

	log *logrus.Entry
)

func init() {
	config.OnInit(func() {
		log = logger.WithField("pkg", "docker").
			WithField("ip", utils.GetHostIP()).
			WithField("mode", config.Mode.String()).
			WithField("user", config.User.String())
		if !config.Mode.IsServer { // docker is only enabled in server mode
			return
		}
		var endpoint string
		for _, loc := range config.Docker.Endpoints {
			if com.IsFile(loc) {
				endpoint = loc
				break
			}
		}

		if endpoint == "" {
			log.WithField("endpoints", spew.Sprint(config.Docker.Endpoints)).Fatal("Cannot find any docker endpoint")
			return
		}

		client, err := NewClient()
		if err != nil {
			err = Error(err)
			log.WithError(err).WithField("endpoint", endpoint).Fatal("Cannot connect to docker endpoint.")
			return
		}

		// get the version of the docker remote API
		env, err := client.Version()
		if err != nil {
			log.WithError(err).WithField("endpoint", endpoint).Fatal("Cannot get version to docker endpoint.")
			return
		}
		log.WithField("version", env.Get("Version")).Debug("Connected to docker client.")

		err = client.GetImage(config.Docker.Image, config.Docker.Tag)
		if err != nil {
			log.WithError(err).
				WithField("image", config.Docker.Image).
				WithField("tag", config.Docker.Tag).
				Error("Failed to pull docker image")
			return
		}

		hostConfig = &dc.HostConfig{
			Privileged:      true,
			AutoRemove:      false,
			PublishAllPorts: false,
			ReadonlyRootfs:  false,
			Memory:          int64(config.Docker.MemoryLimit),
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
			MemorySwap: -1,
			Binds: []string{
				"nvidia_driver_367.57:/usr/local/nvidia:ro",
			},
			Devices: []dc.Device{
				dc.Device{
					PathOnHost:        DeviceCtl,
					PathInContainer:   DeviceCtl,
					CgroupPermissions: "rwm",
				},
				dc.Device{
					PathOnHost:        DeviceUVM,
					PathInContainer:   DeviceUVM,
					CgroupPermissions: "rwm",
				},
				dc.Device{
					PathOnHost:        DeviceUVMTools,
					PathInContainer:   DeviceUVMTools,
					CgroupPermissions: "rwm",
				},
				dc.Device{
					PathOnHost:        "/dev/nvidia0",
					PathInContainer:   "/dev/nvidia0",
					CgroupPermissions: "rwm",
				},
			},
		}
	})
}
