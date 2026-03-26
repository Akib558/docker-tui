package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Created time.Time
	Ports   []PortBinding
	Mounts  []MountInfo
	Network map[string]NetworkInfo
	Env     []string
	Labels  map[string]string
	Command string
	SizeRw  int64
	SizeRootFs int64
	Platform   string
	RestartCount int
}

type PortBinding struct {
	HostIP   string
	HostPort string
	ContPort string
	Protocol string
}

type MountInfo struct {
	Source      string
	Destination string
	Mode        string
	Type        string
	RW          bool
}

type NetworkInfo struct {
	IPAddress  string
	Gateway    string
	MacAddress string
}

type Client struct {
	cli *client.Client
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) ListContainers() ([]ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		var ports []PortBinding
		for _, p := range c.Ports {
			ports = append(ports, PortBinding{
				HostIP:   p.IP,
				HostPort: fmt.Sprintf("%d", p.PublicPort),
				ContPort: fmt.Sprintf("%d", p.PrivatePort),
				Protocol: p.Type,
			})
		}

		var mounts []MountInfo
		for _, m := range c.Mounts {
			mounts = append(mounts, MountInfo{
				Source:      m.Source,
				Destination: m.Destination,
				Mode:        m.Mode,
				Type:        string(m.Type),
				RW:          m.RW,
			})
		}

		networks := make(map[string]NetworkInfo)
		if c.NetworkSettings != nil {
			for name, net := range c.NetworkSettings.Networks {
				networks[name] = NetworkInfo{
					IPAddress:  net.IPAddress,
					Gateway:    net.Gateway,
					MacAddress: net.MacAddress,
				}
			}
		}

		result = append(result, ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
			Ports:   ports,
			Mounts:  mounts,
			Network: networks,
			Labels:  c.Labels,
			Command: c.Command,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].State == result[j].State {
			return result[i].Name < result[j].Name
		}
		if result[i].State == "running" {
			return true
		}
		return false
	})

	return result, nil
}

func (c *Client) InspectContainer(id string) (*ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	name := strings.TrimPrefix(info.Name, "/")

	var ports []PortBinding
	if info.NetworkSettings != nil {
		for port, bindings := range info.NetworkSettings.Ports {
			for _, b := range bindings {
				ports = append(ports, PortBinding{
					HostIP:   b.HostIP,
					HostPort: b.HostPort,
					ContPort: port.Port(),
					Protocol: port.Proto(),
				})
			}
		}
	}

	var mounts []MountInfo
	for _, m := range info.Mounts {
		mounts = append(mounts, MountInfo{
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
			Type:        string(m.Type),
			RW:          m.RW,
		})
	}

	networks := make(map[string]NetworkInfo)
	if info.NetworkSettings != nil {
		for name, net := range info.NetworkSettings.Networks {
			networks[name] = NetworkInfo{
				IPAddress:  net.IPAddress,
				Gateway:    net.Gateway,
				MacAddress: net.MacAddress,
			}
		}
	}

	var env []string
	if info.Config != nil {
		env = info.Config.Env
	}

	platform := info.Platform
	restartCount := info.RestartCount

	return &ContainerInfo{
		ID:           info.ID[:12],
		Name:         name,
		Image:        info.Config.Image,
		Status:       info.State.Status,
		State:        info.State.Status,
		Created:      parseDockerTime(info.Created),
		Ports:        ports,
		Mounts:       mounts,
		Network:      networks,
		Env:          env,
		Labels:       info.Config.Labels,
		Command:      strings.Join(info.Config.Cmd, " "),
		Platform:     platform,
		RestartCount: restartCount,
	}, nil
}

func (c *Client) StartContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *Client) StopContainer(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return c.cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (c *Client) GetContainerLogs(id string, lines int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tail := fmt.Sprintf("%d", lines)
	reader, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer reader.Close()

	buf := make([]byte, 64*1024)
	var result strings.Builder
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return result.String(), nil
}

func parseDockerTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c *Client) GetDockerInfo() (system.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.cli.Info(ctx)
}
