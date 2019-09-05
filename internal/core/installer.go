package core

import (
	"protos/internal/capability"
	"protos/internal/util"
)

// InstallerMetadata holds metadata for the installer
type InstallerMetadata struct {
	Params          []string                 `json:"params"`
	Provides        []string                 `json:"provides"`
	Requires        []string                 `json:"requires"`
	PublicPorts     []util.Port              `json:"publicports"`
	Description     string                   `json:"description"`
	PlatformID      string                   `json:"platformid"`
	PlatformType    string                   `json:"platformtype"`
	PersistancePath string                   `json:"persistancepath"`
	Capabilities    []*capability.Capability `json:"capabilities"`
}
