package docker

type ContainerOptions struct {
	WorkingDirectory string
}

type ContainerOption func(*ContainerOptions)

func WorkingDirectory(dir string) ContainerOption {
	return func(o *ContainerOptions) {
		o.WorkingDirectory = dir
	}
}

func CUDAVolume() ContainerOption {
	return func(o *ContainerOptions) {
		//...
	}
}
