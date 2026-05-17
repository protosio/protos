package runtime

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
