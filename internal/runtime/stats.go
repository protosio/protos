package runtime

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
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

// MemoryInfo holds information bout memory usage
type MemoryInfo struct {
	Total     int `json:"total"`
	Usage     int `json:"usage"`
	Cached    int `json:"cached"`
	Available int `json:"available"`
}

// StorageStats holds information about disk usage
type StorageStats struct {
	Total     int    `json:"total"`
	Path      string `json:"path"`
	Usage     int    `json:"usage"`
	Available int    `json:"available"`
}

// HardwareStats holds information about the state and usage of the system
type HardwareStats struct {
	Memory  MemoryInfo   `json:"memory"`
	CPU     CPUStats     `json:"cpu"`
	Storage StorageStats `json:"storage"`
}

func getHWStatus() (HardwareStats, error) {
	hw := HardwareStats{}

	memDetailedStat, err := mem.VirtualMemory()
	if err != nil {
		return hw, err
	}
	memStat := MemoryInfo{
		Total:     int(memDetailedStat.Total / 1000000),
		Usage:     int(memDetailedStat.UsedPercent),
		Cached:    int(memDetailedStat.Cached / 1000000),
		Available: int(memDetailedStat.Available / 1000000),
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

	// diskStat, err := disk.Usage("/")
	// if err != nil {
	// 	return hw, err
	// }
	storageStat := StorageStats{
		// Total:     int(diskStat.Total / 1000000),
		// Path:      "/",
		// Usage:     int(diskStat.UsedPercent),
		// Available: int(diskStat.Free / 1000000),
	}

	hw.CPU = cpuStat
	hw.Memory = memStat
	hw.Storage = storageStat

	return hw, nil
}
