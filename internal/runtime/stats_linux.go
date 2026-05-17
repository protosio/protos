//go:build linux

package runtime

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

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

	storageStat := StorageStats{}

	hw.CPU = cpuStat
	hw.Memory = memStat
	hw.Storage = storageStat

	return hw, nil
}
