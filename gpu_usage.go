package docker

import (
	"fmt"

	"github.com/flyaways/golang-lru"
	"github.com/pkg/errors"
	"github.com/rai-project/nvidia-smi"
)

type GPUUsageState struct {
	*lru.Cache
}

var GPUDeviceUsageState *GPUUsageState

func NewGPUUsageState() (*GPUUsageState, error) {
	smi := nvidiasmi.Info
	if smi == nil {
		return nil, errors.New("no gpu found")
	}
	cache, err := lru.New(nvidiasmi.GPUCount)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create lru cache")
	}
	for n := 0; n < nvidiasmi.HyperQ; n++ {
		for ii := range smi.GPUS {
			key := fmt.Sprintf("dev[%v];hyperq[%v]", ii, n)
			cache.Add(key, ii)
		}
	}
	return &GPUUsageState{
		Cache: cache,
	}, nil
}
