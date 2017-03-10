package main

import (
	"github.com/rai-project/config"
	"github.com/rai-project/docker/cuda"
)

func main() {
	config.Init()
	cuda.Serve()
}
