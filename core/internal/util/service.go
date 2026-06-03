package util

// ServiceStatus represents the status of a service
type ServiceStatus string

const (
	// StatusActive denotes an active service
	StatusActive = ServiceStatus("active")
	// StatusInactive denotes an inactive service
	StatusInactive = ServiceStatus("inactive")
)

// Service represents a public protos service that responds on a specific domain and ports
type Service struct {
	Ports  []Port        `json:"ports"`
	Domain string        `json:"domain"`
	IP     string        `json:"ip"`
	Name   string        `json:"name"`
	Status ServiceStatus `json:"status"`
}
