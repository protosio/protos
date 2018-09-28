package provider

import (
	"encoding/gob"
	"errors"

	"github.com/protosio/protos/daemon"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/util"
)

var log = util.GetLogger()

//
// Provider code
//

// Provider defines a Protos resource provider
type Provider struct {
	Type  resource.RType `storm:"id"`
	AppID string
}

var providers = make(map[resource.RType]*Provider)

func init() {
	providers[resource.DNS] = &Provider{Type: resource.DNS}
	providers[resource.Certificate] = &Provider{Type: resource.Certificate}
	providers[resource.Mail] = &Provider{Type: resource.Mail}
}

// Register registers a resource provider
func Register(app *daemon.App, rtype resource.RType) error {
	if providers[rtype].AppID != "" {
		_, err := daemon.ReadApp(providers[rtype].AppID)
		if err == nil {
			return errors.New("Provider already registered for resource type " + string(rtype))
		}
	}

	log.Info("Registering provider for resource " + string(rtype))
	providers[rtype].AppID = app.ID
	err := database.Save(providers[rtype])
	if err != nil {
		log.Panicf("Failed to save provider to db: %s", err.Error())
	}

	return nil
}

// Deregister deregisters a resource provider
func Deregister(app *daemon.App, rtype resource.RType) error {

	if providers[rtype].AppID != "" && providers[rtype].AppID != app.ID {
		return errors.New("Application '" + app.Name + "' is NOT registered for resource type " + string(rtype))
	}

	log.Infof("Deregistering application %s(%s) as a provider for %s", app.Name, app.ID, string(rtype))
	providers[rtype].AppID = ""
	err := database.Save(providers[rtype])
	if err != nil {
		log.Panicf("Failed to save provider to db: %s", err.Error())
	}
	return nil
}

// Get retrieves the resource provider associated with an app
func Get(app *daemon.App) (*Provider, error) {
	for _, provider := range providers {
		if provider.AppID != "" && provider.AppID == app.ID {
			return provider, nil
		}
	}
	return nil, errors.New("Application '" + app.Name + "' is NOT a resource provider")
}

// LoadProvidersDB loads the providers from the database
func LoadProvidersDB() {
	log.Info("Retrieving providers from DB")
	gob.Register(&Provider{})

	prvs := []Provider{}
	err := database.All(&prvs)
	if err != nil {
		log.Fatalf("Could not retrieve providers from the database: %s", err.Error())
	}
	for idx, provider := range prvs {
		providers[provider.Type] = &prvs[idx]
	}
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
