
package docker

import (
	"fmt"
	"path/filepath"

	"bitbucket.org/hwuligans/rai/pkg/config"
	"bitbucket.org/hwuligans/rai/pkg/utils"
	"bitbucket.org/hwuligans/rai/pkg/uuid"
)

var (
	ContainerSourceDir = filepath.Join("/src")
	ContainerDataDir   = filepath.Join("/data")
	ContainerBuildDir  = filepath.Join("/build")
)

func baseEnvs() map[string]string {

	// ContainerSourceDir := filepath.Join("/home", userName, "src")
	// ContainerDataDir := filepath.Join("/home", userName, "data")
	// ContainerBuildDir := filepath.Join("/home", userName, "build")

	userName := config.Docker.UserName
	return map[string]string{
		"CI":             "rai",
		"RAI":            "true",
		"RAI_ARCH":       "linux/amd64",
		"RAI_USER":       userName,
		"RAI_SOURCE_DIR": ContainerSourceDir,
		"RAI_DATA_DIR":   ContainerDataDir,
		"RAI_BUILD_DIR":  ContainerBuildDir,
		"DATA_DIR":       ContainerDataDir,
		"SOURCE_DIR":     ContainerSourceDir,
		"BUILD_DIR":      ContainerBuildDir,
		"BUILD_ID":       fmt.Sprintf("%s-build-%s", config.App.Name, uuid.NewV4()),
		"RAI_HOST_IP":    utils.GetHostIP(),
		"TERM":           "xterm",
		"PATH":           "/usr/local/nvidia/bin:/usr/local/cuda/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}
}

func baseEnvsStringList() []string {
	res := []string{}
	for k, v := range baseEnvs() {
		res = append(res, k+"="+v)
	}
	return res
}
