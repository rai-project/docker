package docker

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
)

func Error(err error) error {
	if _, ok := err.(*docker.Error); ok {
		return fmt.Errorf("Docker: %v", err.(*docker.Error).Message)
	}
	return err
}
