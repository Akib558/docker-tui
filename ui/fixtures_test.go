package ui

import (
	"time"

	"github.com/akib558/docker-tui/config"
	"github.com/akib558/docker-tui/docker"
)

// fixtureBase is the timestamp every fixture object is anchored to, so that
// rendered ages and dates are stable.
var fixtureBase = time.Date(2026, 3, 14, 6, 0, 0, 0, time.UTC)

// fixtureContainers covers every state the list renders differently, plus a
// CJK name, an emoji name, and a name long enough to force truncation.
func fixtureContainers() []docker.ContainerInfo {
	return []docker.ContainerInfo{
		{
			ID: "a1b2c3d4e5f6", Name: "web-api", Image: "nginx:1.25-alpine",
			State: "running", Status: "Up 3 hours", Health: "healthy",
			Created: fixtureBase, StartedAt: fixtureBase.Add(20 * time.Minute),
			Ports: []docker.PortBinding{{HostPort: "8080", ContPort: "80/tcp"}},
		},
		{
			ID: "b2c3d4e5f6a1", Name: "postgres-primary", Image: "postgres:16",
			State: "running", Status: "Up 2 days", Health: "unhealthy",
			Created: fixtureBase, StartedAt: fixtureBase.Add(10 * time.Minute),
			Ports: []docker.PortBinding{{HostPort: "5432", ContPort: "5432/tcp"}},
		},
		{
			ID: "c3d4e5f6a1b2", Name: "日本語コンテナ", Image: "redis:7",
			State: "paused", Status: "Paused", Created: fixtureBase,
		},
		{
			ID: "d4e5f6a1b2c3", Name: "worker-🚀-emoji", Image: "python:3.12-slim",
			State: "exited", Status: "Exited (0) 5 minutes ago", Created: fixtureBase,
		},
		{
			ID: "e5f6a1b2c3d4", Name: "this-is-a-deliberately-very-long-container-name-for-truncation",
			Image: "ghcr.io/example/some-quite-long-image-name:v2.11.0",
			State: "created", Status: "Created", Created: fixtureBase,
		},
		{
			ID: "f6a1b2c3d4e5", Name: "zombie", Image: "busybox:latest",
			State: "dead", Status: "Dead", Created: fixtureBase,
		},
	}
}

func fixtureStats() map[string]*docker.ContainerResourceStats {
	return map[string]*docker.ContainerResourceStats{
		"a1b2c3d4e5f6": {CPUPercent: 12.5, MemPercent: 41.0, MemUsage: 429496729, MemLimit: 1073741824, NetRx: 1048576, NetTx: 524288, BlockRead: 2097152, BlockWrite: 1048576},
		"b2c3d4e5f6a1": {CPUPercent: 87.25, MemPercent: 92.5, MemUsage: 993078476, MemLimit: 1073741824, NetRx: 10485760, NetTx: 5242880},
		"c3d4e5f6a1b2": {CPUPercent: 0, MemPercent: 0},
	}
}

func fixtureImages() []docker.ImageInfo {
	return []docker.ImageInfo{
		{ID: "sha256:aaaa1111", Tags: []string{"nginx:1.25-alpine"}, Size: 42_000_000, Created: fixtureBase, Containers: 1},
		{ID: "sha256:bbbb2222", Tags: []string{"postgres:16"}, Size: 431_000_000, Created: fixtureBase, Containers: 1},
		{ID: "sha256:cccc3333", Tags: []string{"<none>:<none>"}, Size: 7_300_000, Created: fixtureBase},
		{ID: "sha256:dddd4444", Tags: []string{"ghcr.io/example/some-quite-long-image-name:v2.11.0"}, Size: 1_200_000_000, Created: fixtureBase},
	}
}

func fixtureVolumes() []docker.VolumeInfo {
	return []docker.VolumeInfo{
		{Name: "pgdata", Driver: "local", Mountpoint: "/var/lib/docker/volumes/pgdata/_data", Scope: "local", CreatedAt: fixtureBase},
		{Name: "app-cache", Driver: "local", Mountpoint: "/var/lib/docker/volumes/app-cache/_data", Scope: "local", CreatedAt: fixtureBase},
		{Name: "a-volume-with-a-really-long-name-that-must-truncate", Driver: "local", Mountpoint: "/var/lib/docker/volumes/a-volume-with-a-really-long-name-that-must-truncate/_data", Scope: "local", CreatedAt: fixtureBase},
	}
}

func fixtureNetworks() []docker.NetworkResource {
	return []docker.NetworkResource{
		{ID: "net1111", Name: "bridge", Driver: "bridge", Scope: "local", Created: fixtureBase},
		{ID: "net2222", Name: "host", Driver: "host", Scope: "local", Created: fixtureBase},
		{ID: "net3333", Name: "compose_default", Driver: "bridge", Scope: "local", Created: fixtureBase, Internal: true},
	}
}

func fixtureEvents() []docker.DockerEvent {
	return []docker.DockerEvent{
		{Time: fixtureBase, Type: "container", Action: "start", Actor: "web-api", ID: "a1b2c3d4e5f6"},
		{Time: fixtureBase.Add(time.Minute), Type: "image", Action: "pull", Actor: "nginx:1.25-alpine", ID: "sha256:aaaa1111"},
		{Time: fixtureBase.Add(2 * time.Minute), Type: "volume", Action: "create", Actor: "pgdata", ID: "pgdata"},
	}
}

func fixtureNotifications() []Notification {
	return []Notification{
		{Message: "Started web-api", Timestamp: fixtureBase},
		{Message: "Failed to remove volume pgdata: volume is in use", IsError: true, Timestamp: fixtureBase.Add(time.Minute)},
		{Message: "Pulled nginx:1.25-alpine", Timestamp: fixtureBase.Add(2 * time.Minute)},
	}
}

func fixtureClient() *docker.FakeClient {
	return &docker.FakeClient{
		Containers: fixtureContainers(),
		Images:     fixtureImages(),
		Volumes:    fixtureVolumes(),
		Networks:   fixtureNetworks(),
	}
}

func fixtureConfig() *config.Config {
	return &config.Config{HintDismissed: true}
}
