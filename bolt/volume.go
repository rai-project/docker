package store

import (
	"sync"

	"github.com/boltdb/bolt"
	dvolume "github.com/docker/go-plugins-helpers/volume"
)

type volume struct {
	sync.Mutex
	db   *bolt.DB
	path string
}

func New(path string) (*volume, error) {

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	return &volume{db: db, path: path}, nil
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
