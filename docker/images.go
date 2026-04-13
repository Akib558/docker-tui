package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
)

type ImageInfo struct {
	ID         string
	Tags       []string
	Size       int64
	Created    time.Time
	Containers int64
}

func (img ImageInfo) DisplayTag() string {
	if len(img.Tags) == 0 {
		return "<none>:<none>"
	}
	return img.Tags[0]
}

func (c *Client) ListImages() ([]ImageInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := c.cli.ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	result := make([]ImageInfo, 0, len(list))
	for _, img := range list {
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:]
		}
		if len(id) > 12 {
			id = id[:12]
		}
		result = append(result, ImageInfo{
			ID:         id,
			Tags:       img.RepoTags,
			Size:       img.Size,
			Created:    time.Unix(img.Created, 0),
			Containers: img.Containers,
		})
	}
	return result, nil
}

func (c *Client) RemoveImage(id string, force bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := c.cli.ImageRemove(ctx, id, image.RemoveOptions{Force: force})
	return err
}

func (c *Client) PullImage(ref string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	reader, err := c.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull %s: %w", ref, err)
	}
	defer reader.Close()

	// Drain the response to ensure pull completes
	io.Copy(io.Discard, reader)
	return nil
}
