package docker

import (
	"testing"
	"time"
)

func TestVolumeInfoDisplayName(t *testing.T) {
	v := VolumeInfo{Name: "my-volume", Driver: "local", Scope: "local", CreatedAt: time.Now()}
	if v.DisplayName() != "my-volume" {
		t.Errorf("expected my-volume, got %s", v.DisplayName())
	}
}

func TestVolumeInfoDisplayNameEmpty(t *testing.T) {
	v := VolumeInfo{}
	if v.DisplayName() != "" {
		t.Errorf("expected empty string, got %s", v.DisplayName())
	}
}

func TestClient_ListVolumes(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Skip("docker not available")
	}
	vols, err := c.ListVolumes()
	if err != nil {
		t.Fatalf("ListVolumes failed: %v", err)
	}
	_ = vols
}