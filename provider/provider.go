package provider

import (
	"errors"

	"github.com/nustiueudinastea/protos/daemon"
	"github.com/nustiueudinastea/protos/resource"
	"github.com/nustiueudinastea/protos/util"
)

var log = util.Log

//
// Provider code
//

// Provider defines a Protos resource provider
type Provider struct {
	Type resource.RType
	App  *daemon.App
}

var providers = make(map[resource.RType]*Provider)

func init() {
	providers[resource.DNS] = &Provider{Type: resource.DNS, App: nil}
	providers[resource.Certificate] = &Provider{Type: resource.Certificate, App: nil}
	providers[resource.Mail] = &Provider{Type: resource.Mail, App: nil}
}

// Register registers a resource provider
func Register(app *daemon.App, rtype resource.RType) error {
	if providers[rtype].App != nil {
		err := errors.New("Provider already registered for resource type " + string(rtype))
		return err
	}

	log.Info("Registering  provider for resource " + string(rtype))
	providers[rtype].App = app
	app.SetProvider(providers[rtype])

	return nil
}

// Deregister deregisters a resource provider
func Deregister(app *daemon.App, rtype resource.RType) error {

	if providers[rtype].App != nil && providers[rtype].App.ID != app.ID {
		return errors.New("Application '" + app.Name + "' is NOT registered for resource type " + string(rtype))
	}

	log.Info("Deregistering application '" + app.Name + "' as a provider for " + string(rtype))
	providers[rtype].App = nil
	return nil
}

//
// Instance methods
//

//GetResources retrieves all resources of a specific resource provider.
func (provider *Provider) GetResources() map[string]*resource.Resource {
	res := map[string]*resource.Resource{}
	for _, resource := range resource.GetForType(provider.Type) {
		res[resource.ID] = resource
	}
	return res
}

//GetResource retrieves a resource that belongs to this provider
func (provider *Provider) GetResource(resourceID string) *resource.Resource {
	for _, resource := range resource.GetForType(provider.Type) {
		if resource.ID == resourceID {
			return resource
		}
	}
	return nil
}

//TypeName returns the name of the type of resource the provider provides
func (provider *Provider) TypeName() string {
	return string(provider.Type)
}
