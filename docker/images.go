package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/docker/docker/api/types/filters"
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
	return c.PullImageWithProgress(ref, nil)
}

func (c *Client) PullImageWithProgress(ref string, onProgress func(string)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	reader, err := c.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull %s: %w", ref, err)
	}
	defer reader.Close()

	if onProgress == nil {
		_, _ = io.Copy(io.Discard, reader)
		return nil
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	last := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var payload struct {
			Status         string `json:"status"`
			ID             string `json:"id"`
			Error          string `json:"error"`
			Progress       string `json:"progress"`
			ProgressDetail struct {
				Current int64 `json:"current"`
				Total   int64 `json:"total"`
			} `json:"progressDetail"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			continue
		}
		if payload.Error != "" {
			return fmt.Errorf("pull error: %s", payload.Error)
		}

		msg := payload.Status
		if payload.ID != "" {
			msg = payload.ID + ": " + msg
		}
		if payload.Progress != "" {
			msg += " " + payload.Progress
		} else if payload.ProgressDetail.Total > 0 {
			pct := int(math.Round(float64(payload.ProgressDetail.Current) * 100 / float64(payload.ProgressDetail.Total)))
			msg += fmt.Sprintf(" %d%%", pct)
		}
		if msg != "" && msg != last {
			last = msg
			onProgress(msg)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func (c *Client) PruneDanglingImages() (ImagePruneResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report, err := c.cli.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		return ImagePruneResult{}, err
	}

	deleted := make([]string, 0, len(report.ImagesDeleted))
	for _, d := range report.ImagesDeleted {
		if d.Deleted != "" {
			deleted = append(deleted, d.Deleted)
		} else if d.Untagged != "" {
			deleted = append(deleted, d.Untagged)
		}
	}

	return ImagePruneResult{DeletedRefs: deleted, SpaceReclaimed: report.SpaceReclaimed}, nil
}
