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

type CUDADriver struct{
  d volume.Driver
}

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

func (CUDADriver) Create(req *volume.CreateRequest) *volume.ErrorResponse {
  vol, version, err := getVolume(req.Name)
	if err != nil {
	  return volume.NewErrorResponse("failed to create " + req.Name + " volume.")
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return volume.NewErrorResponse("volume mismatch " + version + " != " + vol.Version)
	}
	ok, err := vol.Exists()
	if !ok {
		vol.Create(LinkStrategy{})
	}
	return nil
}

func (d *CUDADriver) List() (*volume.ListResponse, error) {
  var req volume.ListResponse

  for _, vol := range VolumeMap {
		versions, err := vol.ListVersions()
		if err != nil {
      return &req, err
			continue
		}
		for _, v := range versions {
			req.Volumes = append(req.Volumes, &volume.Volume{
				Name:       fmt.Sprintf("%s_%s", vol.Name, v),
				Mountpoint: path.Join(vol.Path, v)})
		}
	}
  return &req, nil
}

func (d *CUDADriver) Get(req *volume.GetRequest) (*volume.GetResponse, *volume.ErrorResponse) {
  var gr volume.GetResponse
	vol, version, err := getVolume(req.Name)
  /*gr.Volume = vol*/
	if err != nil {
    return &gr, volume.NewErrorResponse("unable to get volume" +  req.Name)
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return &gr, volume.NewErrorResponse("volume version mismatch" + version + " != " + vol.Version)
	}
	ok, err := vol.Exists(version)
	if err != nil {
		return &gr, volume.NewErrorResponse("unable to check if volume" + vol.Name + "exists")
	}
	if !ok {
		return &gr, volume.NewErrorResponse("volume " + vol.Name + " was not found")
	}
	return &volume.GetResponse{}, nil
}

func (CUDADriver) Remove(req *volume.RemoveRequest) *volume.ErrorResponse {
	vol, version, err := getVolume(req.Name)
	if err != nil {
		return volume.NewErrorResponse("unable to get volume" + req.Name)
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return volume.NewErrorResponse("volume version mismatch " + version + " != " + vol.Version)
	}
	err = vol.Remove(version)
	if err != nil {
		return volume.NewErrorResponse("unable to remove volume " + vol.Name)
	}
	return volume.NewErrorResponse("")
}

func (c CUDADriver) Path(req *volume.PathRequest) (*volume.PathResponse, *volume.ErrorResponse) {
  var pr volume.PathResponse

  mr, err := c.Mount(&volume.MountRequest{Name: req.Name, ID: "0"}) /*Not sure what ID to put here*/
  
	return &pr, volume.NewErrorResponse("")
}

func (CUDADriver) Mount(req *volume.MountRequest) (*volume.MountResponse, *volume.ErrorResponse) {
  var mr volume.MountResponse

	vol, version, err := getVolume(req.Name)
  mr.Mountpoint = vol.Mountpoint
	if err != nil {
		return &mr, volume.NewErrorResponse("unable to get volume " + req.Name)
	}
	// The volume version requested needs to match the volume version in cache
	if version != vol.Version {
		return &mr, volume.NewErrorResponse("volume version mismatch " + version + " != " + vol.Version)
	}
	ok, err := vol.Exists(version)
	if err != nil {
		return &mr, volume.NewErrorResponse("unable to check if volme " + vol.Name +" exists")
	}
	if !ok {
		return &mr, volume.NewErrorResponse("volume " + vol.Name + " was not found")
	}
	return &mr, volume.NewErrorResponse("")
}

func (CUDADriver) Unmount(req *volume.UnmountRequest) *volume.ErrorResponse {
	_, _, err := getVolume(req.Name)
	if err != nil {
		return volume.NewErrorResponse("unable to get volume " + req.Name)
	}
	return volume.NewErrorResponse("")
}

func (CUDADriver) Capabilities() *volume.CapabilitiesResponse {
  var cr volume.CapabilitiesResponse

  cr.Capabilities = volume.Capability{
			Scope: "local",
		}
  return &cr
}

func Serve() {
	d := CUDADriver{}

	h := volume.NewHandler(d.d) /*Not sure how to get driver*/
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
