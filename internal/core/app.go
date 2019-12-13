package core

import (
	"github.com/protosio/protos/internal/util"
)

// WSConnection is a websocket connection via which messages can be sent to the app, if the connection is active
type WSConnection struct {
	Send  chan interface{}
	Close chan bool
}

// AppManager manages applications and their lifecycle
type AppManager interface {
	Read(id string) (App, error)
	GetAllPublic() map[string]App
	Select(func(App) bool) map[string]App
	CreateDevApp(appName string, installerMetadata InstallerMetadata, installerParams map[string]string) (App, error)
	CreateAsync(installerID string, installerVersion string, appName string, installerParams map[string]string, startOnCreation bool) Task
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
	AddTask(id string)
	ValidateCapability(cap Capability) error
	Provides(resourceType string) bool
	ReplaceContainer(id string) error
	AddAction(action string) (Task, error)
	GetResources() map[string]Resource
	GetResource(id string) (Resource, error)
	CreateResource(jsonPayload []byte) (Resource, error)
	DeleteResource(id string) error
	SetMsgQ(msgq *WSConnection)
	CloseMsgQ()
	Public() App
}

// WSPublisher returns a channel that can be used to publish WS messages to the frontend
type WSPublisher interface {
	GetWSPublishChannel() chan interface{}
}
