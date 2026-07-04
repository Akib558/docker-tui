package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/network"
)

type NetworkResource struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Attachable bool
	Labels     map[string]string
	Created    time.Time
}

func (c *Client) ListNetworks() ([]NetworkResource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	result := make([]NetworkResource, 0, len(networks))
	for _, n := range networks {
		result = append(result, NetworkResource{
			ID:         n.ID[:12],
			Name:       n.Name,
			Driver:     n.Driver,
			Scope:      n.Scope,
			Internal:   n.Internal,
			Attachable: n.Attachable,
			Labels:     n.Labels,
			Created:    n.Created,
		})
	}
	return result, nil
}

func (c *Client) RemoveNetwork(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.cli.NetworkRemove(ctx, id)
}
