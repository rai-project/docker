package docker

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/rai-project/cudainfo"
	"github.com/rai-project/ldcache"
)

const (
	binDir   = "bin"
	lib32Dir = "lib"
	lib64Dir = "lib64"
)

type components map[string][]string

type volumeDir struct {
	name  string
	files []string
}

type VolumeInfo struct {
	Name         string
	Mountpoint   string
	MountOptions string
	Components   components
}

type Volume struct {
	*VolumeInfo

	Path    string
	Version string
	dirs    []volumeDir
}

type VolumeMap map[string]*Volume

var Volumes = []VolumeInfo{
	{
		"nvidia_driver",
		"/usr/local/nvidia",
		"ro",
		components{
			"binaries": {
				//"nvidia-modprobe",       // Kernel module loader
				//"nvidia-settings",       // X server settings
				//"nvidia-xconfig",        // X xorg.conf editor
				"nvidia-cuda-mps-control", // Multi process service CLI
				"nvidia-cuda-mps-server",  // Multi process service server
				"nvidia-debugdump",        // GPU coredump utility
				"nvidia-persistenced",     // Persistence mode utility
				"nvidia-smi",              // System management interface
			},
			"libraries": {
				// ------- X11 -------

				//"libnvidia-cfg.so",  // GPU configuration (used by nvidia-xconfig)
				//"libnvidia-gtk2.so", // GTK2 (used by nvidia-settings)
				//"libnvidia-gtk3.so", // GTK3 (used by nvidia-settings)
				//"libnvidia-wfb.so",  // Wrapped software rendering module for X server
				//"libglx.so",         // GLX extension module for X server

				// ----- Compute -----

				"libnvidia-ml.so",              // Management library
				"libcuda.so",                   // CUDA driver library
				"libnvidia-ptxjitcompiler.so",  // PTX-SASS JIT compiler (used by libcuda)
				"libnvidia-fatbinaryloader.so", // fatbin loader (used by libcuda)
				"libnvidia-opencl.so",          // NVIDIA OpenCL ICD
				"libnvidia-compiler.so",        // NVVM-PTX compiler for OpenCL (used by libnvidia-opencl)
				//"libOpenCL.so",               // OpenCL ICD loader

				// ------ Video ------

				"libvdpau_nvidia.so",  // NVIDIA VDPAU ICD
				"libnvidia-encode.so", // Video encoder
				"libnvcuvid.so",       // Video decoder
				"libnvidia-fbc.so",    // Framebuffer capture
				"libnvidia-ifr.so",    // OpenGL framebuffer capture

				// ----- Graphic -----

				// XXX In an ideal world we would only mount nvidia_* vendor specific libraries and
				// install ICD loaders inside the container. However, for backward compatibility reason
				// we need to mount everything. This will hopefully change once GLVND is well established.

				"libGL.so",         // OpenGL/GLX legacy _or_ compatibility wrapper (GLVND)
				"libGLX.so",        // GLX ICD loader (GLVND)
				"libOpenGL.so",     // OpenGL ICD loader (GLVND)
				"libGLESv1_CM.so",  // OpenGL ES v1 common profile legacy _or_ ICD loader (GLVND)
				"libGLESv2.so",     // OpenGL ES v2 legacy _or_ ICD loader (GLVND)
				"libEGL.so",        // EGL ICD loader
				"libGLdispatch.so", // OpenGL dispatch (GLVND) (used by libOpenGL, libEGL and libGLES*)

				"libGLX_nvidia.so",         // OpenGL/GLX ICD (GLVND)
				"libEGL_nvidia.so",         // EGL ICD (GLVND)
				"libGLESv2_nvidia.so",      // OpenGL ES v2 ICD (GLVND)
				"libGLESv1_CM_nvidia.so",   // OpenGL ES v1 common profile ICD (GLVND)
				"libnvidia-eglcore.so",     // EGL core (used by libGLES* or libGLES*_nvidia and libEGL_nvidia)
				"libnvidia-egl-wayland.so", // EGL wayland extensions (used by libEGL_nvidia)
				"libnvidia-glcore.so",      // OpenGL core (used by libGL or libGLX_nvidia)
				"libnvidia-tls.so",         // Thread local storage (used by libGL or libGLX_nvidia)
				"libnvidia-glsi.so",        // OpenGL system interaction (used by libEGL_nvidia)
			},
		},
	},
}

func which(bins ...string) ([]string, error) {
	paths := make([]string, 0, len(bins))

	out, _ := exec.Command("which", bins...).Output()
	r := bufio.NewReader(bytes.NewBuffer(out))
	for {
		p, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if p = strings.TrimSpace(p); !path.IsAbs(p) {
			continue
		}
		path, err := filepath.EvalSymlinks(p)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func LookupVolumes(prefix string) (vols VolumeMap, err error) {
	drv, err := cudainfo.GetDriverVersion()
	if err != nil {
		return nil, err
	}

	cache, err := ldcache.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := cache.Close(); err == nil {
			err = e
		}
	}()

	vols = make(VolumeMap, len(Volumes))

	for i := range Volumes {
		vol := &Volume{
			VolumeInfo: &Volumes[i],
			Path:       path.Join(prefix, Volumes[i].Name),
			Version:    drv,
		}

		for t, c := range vol.Components {
			switch t {
			case "binaries":
				bins, err := which(c...)
				if err != nil {
					return nil, err
				}
				vol.dirs = append(vol.dirs, volumeDir{binDir, bins})
			case "libraries":
				libs32, libs64 := cache.Lookup(c...)
				vol.dirs = append(vol.dirs,
					volumeDir{lib32Dir, libs32},
					volumeDir{lib64Dir, libs64},
				)
			}
		}
		vols[vol.Name] = vol
	}
	return
}
