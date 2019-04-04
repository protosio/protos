package provider

import (
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/protosio/protos/app"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/util"
)

var log = util.GetLogger("provider")

//
// Provider code
//

// Provider defines a Protos resource provider
type Provider struct {
	Type  resource.RType `storm:"id"`
	AppID string
}

type Manager struct {
	providers map[resource.RType]*Provider
}

// var

// func init() {
// 	providers[resource.DNS] = &Provider{Type: resource.DNS}
// 	providers[resource.Certificate] = &Provider{Type: resource.Certificate}
// 	providers[resource.Mail] = &Provider{Type: resource.Mail}
// }

// Register registers a resource provider
func (pm *Manager) Register(appInstance *app.App, rtype resource.RType) error {
	if pm.providers[rtype].AppID != "" {
		if appInstance.ID == pm.providers[rtype].AppID {
			return fmt.Errorf("App %s already registered as a provider for resource type %s", appInstance.ID, string(rtype))
		}

		_, err := app.Read(pm.providers[rtype].AppID)
		if err == nil {
			return errors.New("Another application is registered as a provider for resource type " + string(rtype))
		}
	}

	log.Info("Registering provider for resource " + string(rtype))
	pm.providers[rtype].AppID = appInstance.ID
	err := database.Save(pm.providers[rtype])
	if err != nil {
		log.Panicf("Failed to save provider to db: %s", err.Error())
	}

	return nil
}

// Deregister deregisters a resource provider
func (pm *Manager) Deregister(app *app.App, rtype resource.RType) error {

	if pm.providers[rtype].AppID != "" && pm.providers[rtype].AppID != app.ID {
		return errors.New("Application '" + app.Name + "' is NOT registered for resource type " + string(rtype))
	}

	log.Infof("Deregistering application %s(%s) as a provider for %s", app.Name, app.ID, string(rtype))
	pm.providers[rtype].AppID = ""
	err := database.Save(pm.providers[rtype])
	if err != nil {
		log.Panicf("Failed to save provider to db: %s", err.Error())
	}
	return nil
}

// Get retrieves the resource provider associated with an app
func (pm *Manager) Get(app *app.App) (*Provider, error) {
	for _, provider := range pm.providers {
		if provider.AppID != "" && provider.AppID == app.ID {
			return provider, nil
		}
	}
	return nil, errors.New("Application '" + app.Name + "' is NOT a resource provider")
}

// LoadProvidersDB loads the providers from the database
func (pm *Manager) LoadProvidersDB() {
	log.Info("Retrieving providers from DB")
	gob.Register(&Provider{})

	prvs := []Provider{}
	err := database.All(&prvs)
	if err != nil {
		log.Fatalf("Could not retrieve providers from the database: %s", err.Error())
	}
	for idx, provider := range prvs {
		pm.providers[provider.Type] = &prvs[idx]
	}
}

//
// Instance methods
//

//GetResources retrieves all resources of a specific resource provider.
func (provider *Provider) GetResources() map[string]resource.Resource {
	filter := func(rsc *resource.Resource) bool {
		if rsc.Type == provider.Type {
			return true
		}
		return false
	}
	rscs := resource.Select(filter)
	return rscs
}

//GetResource retrieves a resource that belongs to this provider
func (provider *Provider) GetResource(resourceID string) *resource.Resource {
	rsc, err := resource.Get(resourceID)
	if err != nil {
		log.Error(err)
		return nil
	}
	if rsc.Type != provider.Type {
		log.Errorf("Resource %s is not of type %s, but %s", resourceID, provider.Type, rsc.Type)
		return nil
	}

	return rsc
}

//TypeName returns the name of the type of resource the provider provides
func (provider *Provider) TypeName() string {
	return string(provider.Type)
}
