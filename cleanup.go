package docker

import (
	"context"
	"time"

	"github.com/carlescere/scheduler"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

func cleanupDeadContainers() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	client, err := NewClient(ClientContext(ctx))
	if err != nil {
		return
	}
	defer client.Close()
	imgs, err := client.ImageList(
		ctx,
		types.ImageListOptions{
			All: true,
		},
	)
	if err != nil {
		return
	}
	for _, img := range imgs {
		info, err := client.ContainerInspect(ctx, img.ID)
		if err != nil {
			continue
		}
		if info.State != nil && (info.State.Dead || !info.State.Running) {
			client.ContainerRemove(
				ctx,
				img.ID,
				types.ContainerRemoveOptions{
					Force:         true,
					RemoveVolumes: true,
				},
			)
		}
	}
}

func cleanupDeadVolumes() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	client, err := NewClient(ClientContext(ctx))
	if err != nil {
		return
	}
	defer client.Close()
	vols, err := client.VolumeList(ctx, filters.Args{})
	if err != nil {
		return
	}
	for _, vol := range vols.Volumes {
		client.VolumeRemove(ctx, vol.Name, false)
		client.VolumesPrune(ctx, filters.Args{})
	}
}

func PeriodicCleanupDeadContainers() {
	scheduler.Every(5).Minutes().NotImmediately().Run(cleanupDeadContainers)
	scheduler.Every(10).Minutes().NotImmediately().Run(cleanupDeadVolumes)
}
