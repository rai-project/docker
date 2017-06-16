package cuda

import (
	"fmt"
	"path"
	"regexp"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/k0kubun/pp"
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/pkg/errors"
	nvidiasmi "github.com/rai-project/nvidia-smi"
)

type CUDADriver struct{}

func getVolume(name string) (*Volume, string, error) {
	re := regexp.MustCompile("^([a-zA-Z0-9_.-]+)_([0-9.]+)$")
	m := re.FindStringSubmatch(name)
	if false && len(m) == 2 && nvidiasmi.HasGPU {
		return getVolume(name + "_" + nvidiasmi.Info.DriverVersion)
	}
	if len(m) != 3 {
		return nil, "", errors.Errorf("%v is not a valid volume format", name)
	}
	pp.Println("volume = ", m[1], " version = ", m[2])
	volume, version := VolumeMap[m[1]], m[2]
	if volume == nil {
		return nil, "", errors.Errorf("%v volume is not supported", m[1])
	}
	return volume, version, nil
}

func (CUDADriver) Create(req volume.Request) volume.Response {
	vol, version, err := getVolume(req.Name)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "failed to create %v volume", req.Name).Error(),
		}
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return volume.Response{
			Err: errors.Errorf("volume version mismatch %v != %v", version, vol.Version).Error(),
		}
	}
	ok, err := vol.Exists()
	if !ok {
		vol.Create(LinkStrategy{})
	}
	return volume.Response{}
}

func (CUDADriver) List(req volume.Request) volume.Response {
	var r volume.Response
	for _, vol := range VolumeMap {
		versions, err := vol.ListVersions()
		if err != nil {
			log.WithError(err).Error("failed to get volume %v version information", vol.Name)
			continue
		}
		for _, v := range versions {
			r.Volumes = append(r.Volumes, &volume.Volume{
				Name:       fmt.Sprintf("%s_%s", vol.Name, v),
				Mountpoint: path.Join(vol.Path, v),
			})
		}
	}
	return r
}

func (CUDADriver) Get(req volume.Request) volume.Response {
	vol, version, err := getVolume(req.Name)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to get volume %v", req.Name).Error(),
		}
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return volume.Response{
			Err: errors.Errorf("volume version mismatch %v != %v", version, vol.Version).Error(),
		}
	}
	ok, err := vol.Exists(version)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to check if volme %v exists", vol.Name).Error(),
		}
	}
	if !ok {
		return volume.Response{
			Err: errors.Errorf("volume %v was not found", vol.Name).Error(),
		}
	}
	return volume.Response{
		Volume: &volume.Volume{
			Name:       vol.Name,
			Mountpoint: path.Join(vol.Path, version),
		},
	}
}

func (CUDADriver) Remove(req volume.Request) volume.Response {
	vol, version, err := getVolume(req.Name)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to get volume %v", req.Name).Error(),
		}
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return volume.Response{
			Err: errors.Errorf("volume version mismatch %v != %v", version, vol.Version).Error(),
		}
	}
	err = vol.Remove(version)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to remove volume %v", vol.Name).Error(),
		}
	}
	return volume.Response{}
}

func (c CUDADriver) Path(req volume.Request) volume.Response {
	return c.Mount(volume.MountRequest{Name: req.Name})
}

func (CUDADriver) Mount(req volume.MountRequest) volume.Response {
	vol, version, err := getVolume(req.Name)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to get volume %v", req.Name).Error(),
		}
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return volume.Response{
			Err: errors.Errorf("volume version mismatch %v != %v", version, vol.Version).Error(),
		}
	}
	ok, err := vol.Exists(version)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to check if volme %v exists", vol.Name).Error(),
		}
	}
	if !ok {
		return volume.Response{
			Err: errors.Errorf("volume %v was not found", vol.Name).Error(),
		}
	}
	return volume.Response{
		Mountpoint: path.Join(vol.Path, version),
	}
}

func (CUDADriver) Unmount(req volume.UnmountRequest) volume.Response {
	_, _, err := getVolume(req.Name)
	if err != nil {
		return volume.Response{
			Err: errors.Wrapf(err, "unable to get volume %v", req.Name).Error(),
		}
	}
	return volume.Response{}
}

func (CUDADriver) Capabilities(req volume.Request) volume.Response {
	return volume.Response{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}

func Serve() {
	d := CUDADriver{}

	h := volume.NewHandler(d)
	log.Debug("starting to create a rai-cuda new volume handler")
	gid, err := lookupGidByName("docker")
	if err != nil {
		log.WithError(err).Error("Failed to get gid for docker user")
	}
	log.WithField("gid", gid).Debug("starting rai-cuda docker plugin")
	_ = gid
	err = h.ServeUnix("/run/docker/plugins/rai-cuda.sock", 0)
	if err != nil {
		log.WithError(err).Error("Failed to serve rai-cuda using localhost")
	}
}

func lookupGidByName(group string) (int, error) {
	grp, err := user.LookupGroup(group)
	if err != nil {
		return -1, err
	}
	return grp.Gid, nil
}
