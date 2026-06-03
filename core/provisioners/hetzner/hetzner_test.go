package hetzner

import (
	"strings"
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

func TestSelectHetznerUploadServerTypeRequiresTargetDisk(t *testing.T) {
	_, err := selectHetznerUploadServerType([]*hcloud.ServerType{
		hetznerServerType("cx82", 80, hcloud.ArchitectureX86, "hil", true, "3.00", 8),
	}, "hil")
	if err == nil {
		t.Fatal("expected an error when only oversized upload helpers are available")
	}
	if !strings.Contains(err.Error(), "oversized snapshot") {
		t.Fatalf("error = %q, want oversized snapshot context", err.Error())
	}
}

func TestSelectHetznerUploadServerTypePrefersSmallestTargetDiskBeforePrice(t *testing.T) {
	selected, err := selectHetznerUploadServerType([]*hcloud.ServerType{
		hetznerServerType("cx82", 80, hcloud.ArchitectureX86, "hil", true, "1.00", 2),
		hetznerServerType("cpx11-expensive", 40, hcloud.ArchitectureX86, "hil", true, "9.00", 4),
		hetznerServerType("cpx11-cheap", 40, hcloud.ArchitectureX86, "hil", true, "2.00", 8),
		hetznerServerType("arm-40", 40, hcloud.ArchitectureARM, "hil", true, "0.50", 2),
		hetznerServerType("unavailable-40", 40, hcloud.ArchitectureX86, "hil", false, "0.25", 2),
	}, "hil")
	if err != nil {
		t.Fatalf("selectHetznerUploadServerType() error = %v", err)
	}
	if selected.Name != "cpx11-cheap" {
		t.Fatalf("selected %q, want cpx11-cheap", selected.Name)
	}
}

func hetznerServerType(name string, disk int, arch hcloud.Architecture, location string, available bool, monthly string, memory float32) *hcloud.ServerType {
	loc := &hcloud.Location{Name: location}
	return &hcloud.ServerType{
		Name:         name,
		Disk:         disk,
		Memory:       memory,
		Architecture: arch,
		Locations: []hcloud.ServerTypeLocation{
			{Location: loc, Available: available},
		},
		Pricings: []hcloud.ServerTypeLocationPricing{
			{Location: loc, Monthly: hcloud.Price{Net: monthly}},
		},
	}
}
