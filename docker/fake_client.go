// Package docker provides a thin client wrapper around the Docker Engine API.
package docker

import (
	"context"
	"io"
)

// FakeClient is a minimal ClientAPI stub for tests.
type FakeClient struct {
	Containers []ContainerInfo
	Volumes    []VolumeInfo
	Networks   []NetworkResource
	Images     []ImageInfo
}

func (f *FakeClient) ListContainers() ([]ContainerInfo, error) {
	return f.Containers, nil
}
func (f *FakeClient) InspectContainer(id string) (*ContainerInfo, error) { return nil, nil }
func (f *FakeClient) StartContainer(id string) error                     { return nil }
func (f *FakeClient) StopContainer(id string) error                      { return nil }
func (f *FakeClient) RestartContainer(id string) error                   { return nil }
func (f *FakeClient) PauseContainer(id string) error                     { return nil }
func (f *FakeClient) UnpauseContainer(id string) error                   { return nil }
func (f *FakeClient) KillContainer(id, signal string) error              { return nil }
func (f *FakeClient) RemoveContainer(id string, force bool) error        { return nil }
func (f *FakeClient) GetContainerDiff(id string) ([]DiffEntry, error)    { return nil, nil }
func (f *FakeClient) GetContainerTop(id string) (ContainerTop, error)    { return ContainerTop{}, nil }
func (f *FakeClient) GetContainerLogs(id string, lines int) (string, error) {
	return "", nil
}
func (f *FakeClient) GetContainerLogRecords(id string, lines int) ([]LogRecord, error) {
	return nil, nil
}
func (f *FakeClient) GetContainerLogsStream(ctx context.Context, id string) (io.ReadCloser, error) {
	return nil, nil
}
func (f *FakeClient) GetContainerLogRecordsStream(ctx context.Context, id string, tail int) (io.ReadCloser, error) {
	return nil, nil
}
func (f *FakeClient) StartContainerExecShell(ctx context.Context, id, shell string) (io.ReadWriteCloser, error) {
	return nil, nil
}
func (f *FakeClient) GetContainerStats(id string) (*ContainerResourceStats, error) { return nil, nil }
func (f *FakeClient) GetAllContainerStats(ids []string) map[string]*ContainerResourceStats {
	return nil
}
func (f *FakeClient) ListImages() ([]ImageInfo, error)        { return f.Images, nil }
func (f *FakeClient) RemoveImage(id string, force bool) error { return nil }
func (f *FakeClient) PullImage(ref string) error              { return nil }
func (f *FakeClient) PullImageWithProgress(ref string, onProgress func(string)) error {
	return nil
}
func (f *FakeClient) PruneDanglingImages() (ImagePruneResult, error) { return ImagePruneResult{}, nil }
func (f *FakeClient) ListNetworks() ([]NetworkResource, error)       { return f.Networks, nil }
func (f *FakeClient) RemoveNetwork(id string) error                  { return nil }
func (f *FakeClient) ListVolumes() ([]VolumeInfo, error)             { return f.Volumes, nil }
func (f *FakeClient) RemoveVolume(name string) error                 { return nil }
func (f *FakeClient) PruneVolumes() ([]string, error)                { return nil, nil }
func (f *FakeClient) SystemPrune() (SystemPruneResult, error)        { return SystemPruneResult{}, nil }
func (f *FakeClient) StreamEvents(ctx context.Context) <-chan DockerEvent {
	ch := make(chan DockerEvent)
	close(ch)
	return ch
}
func (f *FakeClient) GetDockerOverview() (*DockerOverview, error) { return &DockerOverview{}, nil }
func (f *FakeClient) Close() error                                { return nil }

var _ ClientAPI = (*FakeClient)(nil)
