package core

// // InstallerMetadata holds metadata for the installer
// type InstallerMetadata struct {
// 	Params          []string    `json:"params"`
// 	Provides        []string    `json:"provides"`
// 	Requires        []string    `json:"requires"`
// 	PublicPorts     []util.Port `json:"publicports"`
// 	Description     string      `json:"description"`
// 	PlatformID      string      `json:"platformid"`
// 	PlatformType    string      `json:"platformtype"`
// 	PersistancePath string      `json:"persistancepath"`
// 	Capabilities    []string    `json:"capabilities"`
// }

// // AppStore manages and downloads application installers
// type AppStore interface {
// 	GetInstallers() (map[string]Installer, error)
// 	GetInstaller(id string) (Installer, error)
// 	Search(key string, value string) (map[string]Installer, error)
// }

// // Installer represents a Protos installed
// type Installer interface {
// 	GetName() string
// 	GetLastVersion() string
// 	GetMetadata(version string) (InstallerMetadata, error)
// 	IsPlatformImageAvailable(version string) (bool, error)
// 	DownloadAsync(version string, appID string) Task
// }
