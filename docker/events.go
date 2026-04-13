package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
)

type DockerEvent struct {
	Time   time.Time
	Type   string
	Action string
	Actor  string
	ID     string
}

// StreamEvents returns a channel of DockerEvents and a cancel func.
// Close the returned cancel to stop the stream.
func (c *Client) StreamEvents(ctx context.Context) <-chan DockerEvent {
	out := make(chan DockerEvent, 200)

	go func() {
		defer close(out)

		f := filters.NewArgs(filters.Arg("type", "container"))
		msgCh, errCh := c.cli.Events(ctx, events.ListOptions{Filters: f})

		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errCh:
				if err != nil {
					return
				}
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				name := msg.Actor.Attributes["name"]
				out <- DockerEvent{
					Time:   time.Unix(msg.Time, 0),
					Type:   string(msg.Type),
					Action: string(msg.Action),
					Actor:  name,
					ID:     msg.Actor.ID[:min12(msg.Actor.ID)],
				}
			}
		}
	}()

	return out
}

func min12(s string) int {
	if len(s) < 12 {
		return len(s)
	}
	return 12
}
