package store

import (
	"sync"

	dvolume "github.com/docker/go-plugins-helpers/volume"
	"github.com/rai-project/store"
	"github.com/rai-project/store/s3"
)

type volume struct {
	sync.Mutex
	session store.Store
	options store.Options
}

func New(opts ...store.Option) (*volume, error) {
	options := store.Options{}

	for _, o := range opts {
		o(&options)
	}

	sess, err := s3.New(opts...)
	if err != nil {
		return nil, err
	}

	return &volume{session: sess, options: options}, nil
}

func (vol *volume) Create(dvolume.Request) dvolume.Response {
	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) List(dvolume.Request) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) Get(dvolume.Request) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) Remove(dvolume.Request) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) Path(dvolume.Request) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) Mount(dvolume.MountRequest) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) Unmount(dvolume.UnmountRequest) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}

func (vol *volume) Capabilities(dvolume.Request) dvolume.Response {

	vol.Lock()
	defer vol.Unlock()
}
