package platform

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// CPUInfo holds information about the characteristics of the CPU
type CPUInfo struct {
	Model     string  `json:"model"`
	Cores     int     `json:"cores"`
	Frequency float64 `json:"frequency"`
	Cache     int32   `json:"cache"`
}

// CPUStats holds information about the characteristics of the CPU and it's usage
type CPUStats struct {
	Usage int     `json:"usage"`
	Info  CPUInfo `json:"info"`
}

// HardwareStats holds information about the state and usage of the system
type HardwareStats struct {
	Memory  *mem.VirtualMemoryStat `json:"memory"`
	CPU     CPUStats               `json:"cpu"`
	Storage *disk.UsageStat        `json:"storage"`
}

// GetHWStats returns the current system stats
func GetHWStats() (HardwareStats, error) {
	hw := HardwareStats{}

	memStat, err := mem.VirtualMemory()
	if err != nil {
		return hw, err
	}

	cpuDetailedInfo, err := cpu.Info()
	if err != nil {
		return hw, err
	}
	cpuInfo := CPUInfo{
		Model:     cpuDetailedInfo[0].ModelName,
		Cores:     len(cpuDetailedInfo),
		Frequency: cpuDetailedInfo[0].Mhz,
		Cache:     cpuDetailedInfo[0].CacheSize,
	}
	cpuUsage, err := cpu.Percent(time.Second, false)
	if err != nil {
		return hw, err
	}
	cpuStat := CPUStats{Info: cpuInfo, Usage: int(cpuUsage[0])}

	diskStat, err := disk.Usage("/")
	if err != nil {
		return hw, err
	}
	hw.CPU = cpuStat
	hw.Memory = memStat
	hw.Storage = diskStat

	return hw, nil
}
