package docker

import (
	"context"

	"github.com/carlescere/scheduler"
	"github.com/docker/docker/api/types"
	"github.com/rai-project/config"
)

func cleanup() {
	client, err := NewClient()
	if err != nil {
		return
	}
	ctx := context.Background()
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
					Force: true,
				},
			)
		}
	}
}

func init() {
	config.AfterInit(func() {
		scheduler.Every(5).Minutes().NotImmediately().Run(cleanup)
	})
}
