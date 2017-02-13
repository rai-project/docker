package docker

import (
	"github.com/draganm/emission"
	docker "github.com/fsouza/go-dockerclient"
)

type EventEmitter interface {
	Emit(event interface{}, arguments ...interface{}) EventEmitter
	RemoveListener(event, listener interface{}) EventEmitter
	AddListener(event, listener interface{}) EventEmitter
}

var ContainerStats = NewEmitterAdapter()

func NewEmitterAdapter() *EmitterAdapter {
	return &EmitterAdapter{emission.NewEmitter()}
}

type EmitterAdapter struct {
	*emission.Emitter
}

func (e *EmitterAdapter) Emit(event interface{}, arguments ...interface{}) EventEmitter {
	e.Emitter.Emit(event, arguments...)
	return e
}

func (e *EmitterAdapter) RemoveListener(event, listener interface{}) EventEmitter {
	e.Emitter.RemoveListener(event, listener)
	return e
}

func (e *EmitterAdapter) AddListener(event, listener interface{}) EventEmitter {
	e.Emitter.AddListener(event, listener)
	return e
}

func StartTrackingLocalContainerStats(client *docker.Client) {

	containers, err := client.ListContainers(docker.ListContainersOptions{
		All: true,
	})
	if err != nil {
		panic(err)
	}
	for _, c := range containers {
		if c.State == "running" {
			go trackContainer(c.ID, client)
		}
	}

	eventsChannel := make(chan *docker.APIEvents)

	err = client.AddEventListener(eventsChannel)
	if err != nil {
		panic(err)
	}

	go func() {
		for evt := range eventsChannel {
			if evt.Status == "start" {
				go trackContainer(evt.ID, client)
			}
		}
	}()

}

func trackContainer(id string, client *docker.Client) {
	statsChan := make(chan *docker.Stats)

	go func() {
		err := client.Stats(docker.StatsOptions{Stream: true, ID: id, Stats: statsChan})
		if err != nil {
			log.Printf("Error getting status of container %s: %s", id, err)
			return
		}
	}()

	for update := range statsChan {
		ContainerStats.Emit("stats", update)
	}

}
