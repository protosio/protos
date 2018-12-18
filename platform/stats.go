package platform

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

// CPUStats holds information about the characteristics of the CPU and it's usage
type CPUStats struct {
	Usage []cpu.TimesStat `json:"usage"`
	Info  []cpu.InfoStat  `json:"info"`
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
	cpuInfo, err := cpu.Info()
	if err != nil {
		return hw, err
	}
	cpuStats, err := cpu.Times(false)
	if err != nil {
		return hw, err
	}
	cpuStat := CPUStats{Info: cpuInfo, Usage: cpuStats}
	diskStat, err := disk.Usage("/")
	if err != nil {
		return hw, err
	}
	hw.CPU = cpuStat
	hw.Memory = memStat
	hw.Storage = diskStat

	return hw, nil
}
