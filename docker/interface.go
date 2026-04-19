package docker

import (
	"context"
	"io"
)

// ClientAPI defines the Docker operations used by the UI.
// A concrete *Client satisfies this interface; tests can use a mock.
type ClientAPI interface {
	// Container operations
	ListContainers() ([]ContainerInfo, error)
	InspectContainer(id string) (*ContainerInfo, error)
	StartContainer(id string) error
	StopContainer(id string) error
	RestartContainer(id string) error
	RemoveContainer(id string, force bool) error
	GetContainerDiff(id string) ([]DiffEntry, error)
	GetContainerLogs(id string, lines int) (string, error)
	GetContainerLogsStream(ctx context.Context, id string) (io.ReadCloser, error)
	StartContainerExecShell(ctx context.Context, id, shell string) (io.ReadWriteCloser, error)

	// Stats
	GetContainerStats(id string) (*ContainerResourceStats, error)
	GetAllContainerStats(ids []string) map[string]*ContainerResourceStats

	// Images
	ListImages() ([]ImageInfo, error)
	RemoveImage(id string, force bool) error
	PullImage(ref string) error

	// Events
	StreamEvents(ctx context.Context) <-chan DockerEvent

	// System info
	GetDockerOverview() (*DockerOverview, error)

	// Lifecycle
	Close() error
}

// verify *Client satisfies the interface at compile time
var _ ClientAPI = (*Client)(nil)
