package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/volume"
	"time"
)

type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	Scope      string
	CreatedAt  time.Time
}

func (v VolumeInfo) DisplayName() string {
	return v.Name
}

func (c *Client) ListVolumes() ([]VolumeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := c.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	result := make([]VolumeInfo, 0, len(list.Volumes))
	for _, vol := range list.Volumes {
		var createdAt time.Time
		if vol.CreatedAt != "" {
			createdAt, _ = time.Parse(time.RFC3339, vol.CreatedAt)
		}
		result = append(result, VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Labels:     vol.Labels,
			Scope:      vol.Scope,
			CreatedAt:  createdAt,
		})
	}
	return result, nil
}