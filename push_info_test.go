package docker

import "github.com/rai-project/model"

func init() {
	testPushModel = model.Push{
		Push:      true,
		ImageName: "raiproject/zipkin-cpp",
		Registry:  "https://index.docker.io/v1/",
		Credentials: model.Credentials{
			Username: "dakkak",
			Password: "XXXX",
		},
	}
}
