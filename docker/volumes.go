package docker

import (
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