package core

import (
	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/util"
)

type AppManager interface {
	Read(string) (App, error)
	GetAllPublic() map[string]App
	Select(func(App) bool) map[string]App
	CreateDevApp(installerID string, installerVersion string, appName string, installerMetadata InstallerMetadata, installerParams map[string]string) (App, error)
	CreateAsync(installerID string, installerVersion string, appName string, installerMetadata InstallerMetadata, installerParams map[string]string, startOnCreation bool) Task
	GetCopy(id string) (App, error)
	Remove(string) error
	RemoveAsync(string) Task
	GetServices() []util.Service
	CopyAll() map[string]App
}

type App interface {
	Start() error
	Stop() error
	GetID() string
	GetName() string
	GetIP() string
	ValidateCapability(*capability.Capability) error
	Provides(string) bool
	ReplaceContainer(string) error
	AddAction(string) (Task, error)
	CloseMsgQ()
	Public() App
}
