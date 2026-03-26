package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
)

type ContainerInfo struct {
	ID           string
	Name         string
	Image        string
	Status       string
	State        string
	Created      time.Time
	Ports        []PortBinding
	Mounts       []MountInfo
	Network      map[string]NetworkInfo
	Env          []string
	Labels       map[string]string
	Command      string
	Platform     string
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

type ContainerResourceStats struct {
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
	MemPercent float64
	NetRx      uint64
	NetTx      uint64
	BlockRead  uint64
	BlockWrite uint64
	PIDs       uint64
}

type DockerOverview struct {
	ServerVersion string
	Images        int
	TotalMemory   uint64
	CPUs          int
	OS            string
}

// internal: JSON shape returned by Docker stats API
type dockerStatsJSON struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
		Stats struct {
			Cache        uint64 `json:"cache"`
			InactiveFile uint64 `json:"inactive_file"`
		} `json:"stats"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	BlkioStats struct {
		IoServiceBytesRecursive []struct {
			Op    string `json:"op"`
			Value uint64 `json:"value"`
		} `json:"io_service_bytes_recursive"`
	} `json:"blkio_stats"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

// ── Client ──────────────────────────────────────────────────────────────

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

// ── Containers ──────────────────────────────────────────────────────────

func (c *Client) ListContainers() ([]ContainerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, ct := range containers {
		name := ""
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}

		var ports []PortBinding
		for _, p := range ct.Ports {
			ports = append(ports, PortBinding{
				HostIP:   p.IP,
				HostPort: fmt.Sprintf("%d", p.PublicPort),
				ContPort: fmt.Sprintf("%d", p.PrivatePort),
				Protocol: p.Type,
			})
		}

		var mounts []MountInfo
		for _, m := range ct.Mounts {
			mounts = append(mounts, MountInfo{
				Source:      m.Source,
				Destination: m.Destination,
				Mode:        m.Mode,
				Type:        string(m.Type),
				RW:          m.RW,
			})
		}

		networks := make(map[string]NetworkInfo)
		if ct.NetworkSettings != nil {
			for nname, net := range ct.NetworkSettings.Networks {
				networks[nname] = NetworkInfo{
					IPAddress:  net.IPAddress,
					Gateway:    net.Gateway,
					MacAddress: net.MacAddress,
				}
			}
		}

		result = append(result, ContainerInfo{
			ID:      ct.ID[:12],
			Name:    name,
			Image:   ct.Image,
			Status:  ct.Status,
			State:   ct.State,
			Created: time.Unix(ct.Created, 0),
			Ports:   ports,
			Mounts:  mounts,
			Network: networks,
			Labels:  ct.Labels,
			Command: ct.Command,
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
		for nname, net := range info.NetworkSettings.Networks {
			networks[nname] = NetworkInfo{
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
		Platform:     info.Platform,
		RestartCount: info.RestartCount,
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
		n, readErr := reader.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	return result.String(), nil
}

// ── Stats ───────────────────────────────────────────────────────────────

func (c *Client) GetContainerStats(id string) (*ContainerResourceStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.cli.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw dockerStatsJSON
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// CPU percentage
	cpuDelta := float64(raw.CPUStats.CPUUsage.TotalUsage - raw.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(raw.CPUStats.SystemCPUUsage - raw.PreCPUStats.SystemCPUUsage)
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpus := raw.CPUStats.OnlineCPUs
		if cpus == 0 {
			cpus = 1
		}
		cpuPercent = (cpuDelta / sysDelta) * float64(cpus) * 100.0
	}

	// Memory (subtract cache for accurate usage)
	memUsage := raw.MemoryStats.Usage
	cache := raw.MemoryStats.Stats.InactiveFile
	if cache == 0 {
		cache = raw.MemoryStats.Stats.Cache
	}
	if memUsage > cache {
		memUsage -= cache
	}

	memLimit := raw.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}

	// Network I/O
	var netRx, netTx uint64
	for _, net := range raw.Networks {
		netRx += net.RxBytes
		netTx += net.TxBytes
	}

	// Block I/O
	var blkRead, blkWrite uint64
	for _, bio := range raw.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(bio.Op) {
		case "read":
			blkRead += bio.Value
		case "write":
			blkWrite += bio.Value
		}
	}

	return &ContainerResourceStats{
		CPUPercent: cpuPercent,
		MemUsage:   memUsage,
		MemLimit:   memLimit,
		MemPercent: memPercent,
		NetRx:      netRx,
		NetTx:      netTx,
		BlockRead:  blkRead,
		BlockWrite: blkWrite,
		PIDs:       raw.PidsStats.Current,
	}, nil
}

func (c *Client) GetAllContainerStats(ids []string) map[string]*ContainerResourceStats {
	result := make(map[string]*ContainerResourceStats)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 5) // limit concurrency

	for _, id := range ids {
		wg.Add(1)
		sem <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()
			stats, err := c.GetContainerStats(id)
			if err == nil {
				mu.Lock()
				result[id] = stats
				mu.Unlock()
			}
		}(id)
	}

	wg.Wait()
	return result
}

// ── Docker info ─────────────────────────────────────────────────────────

func (c *Client) GetDockerOverview() (*DockerOverview, error) {
	info, err := c.getInfo()
	if err != nil {
		return nil, err
	}
	return &DockerOverview{
		ServerVersion: info.ServerVersion,
		Images:        info.Images,
		TotalMemory:   uint64(info.MemTotal),
		CPUs:          info.NCPU,
		OS:            info.OperatingSystem,
	}, nil
}

func (c *Client) getInfo() (system.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.cli.Info(ctx)
}

func parseDockerTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
