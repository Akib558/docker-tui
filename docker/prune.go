package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/filters"
)

func (c *Client) SystemPrune() (SystemPruneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cont, err := c.cli.ContainersPrune(ctx, filters.NewArgs())
	if err != nil {
		return SystemPruneResult{}, err
	}
	imgs, err := c.cli.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		return SystemPruneResult{}, err
	}
	nets, err := c.cli.NetworksPrune(ctx, filters.NewArgs())
	if err != nil {
		return SystemPruneResult{}, err
	}
	vols, err := c.cli.VolumesPrune(ctx, filters.NewArgs())
	if err != nil {
		return SystemPruneResult{}, err
	}

	return SystemPruneResult{
		ContainersDeleted: len(cont.ContainersDeleted),
		ImagesDeleted:     len(imgs.ImagesDeleted),
		NetworksDeleted:   len(nets.NetworksDeleted),
		VolumesDeleted:    len(vols.VolumesDeleted),
		SpaceReclaimed:    cont.SpaceReclaimed + imgs.SpaceReclaimed + vols.SpaceReclaimed,
	}, nil
}
