package docker

import (
	"github.com/Sirupsen/logrus"
	"github.com/Unknwon/com"
	"github.com/davecgh/go-spew/spew"
	dc "github.com/fsouza/go-dockerclient"

	"github.com/rai-project/config"
	logger "github.com/rai-project/logger"
)

var (
	client *dc.Client

	hostConfig *dc.HostConfig

	log *logrus.Entry
)

func init() {
	config.OnInit(func() {
		log = logger.WithField("pkg", "docker")

		var endpoint string
		for _, loc := range Config.Endpoints {
			if com.IsFile(loc) {
				endpoint = loc
				break
			}
		}

		if endpoint == "" {
			log.WithField("endpoints", spew.Sprint(Config.Endpoints)).Fatal("Cannot find any docker endpoint")
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

		err = client.GetImage(Config.Repository, Config.Tag)
		if err != nil {
			log.WithError(err).
				WithField("image", Config.Repository).
				WithField("tag", Config.Tag).
				Error("Failed to pull docker image")
			return
		}

		hostConfig = &dc.HostConfig{
			Privileged:      true,
			AutoRemove:      false,
			PublishAllPorts: false,
			ReadonlyRootfs:  false,
			Memory:          int64(Config.MemoryLimit),
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
