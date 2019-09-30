package core

import (
	"protos/internal/capability"
	"protos/internal/util"
)

// AppManager manages applications
type AppManager interface {
	Read(id string) (App, error)
	GetAllPublic() map[string]App
	Select(func(App) bool) map[string]App
	CreateDevApp(installerID string, installerVersion string, appName string, installerMetadata InstallerMetadata, installerParams map[string]string) (App, error)
	CreateAsync(installerID string, installerVersion string, appName string, installerMetadata InstallerMetadata, installerParams map[string]string, startOnCreation bool) Task
	GetCopy(id string) (App, error)
	Remove(id string) error
	RemoveAsync(string) Task
	GetServices() []util.Service
	CopyAll() map[string]App
}

// App interface represents an application
type App interface {
	Start() error
	Stop() error
	GetID() string
	GetName() string
	GetIP() string
	AddTask(string)
	ValidateCapability(*capability.Capability) error
	Provides(string) bool
	ReplaceContainer(string) error
	AddAction(string) (Task, error)
	CloseMsgQ()
	Public() App
}

// WSPublisher returns a channel that can be used to publish WS messages to the frontend
type WSPublisher interface {
	GetWSPublishChannel() chan interface{}
}
