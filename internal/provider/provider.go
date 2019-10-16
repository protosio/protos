package provider

import (
	"errors"
	"fmt"

	"protos/internal/core"
	"protos/internal/util"
)

var log = util.GetLogger("provider")

//
// Provider code
//

// Provider defines a Protos resource provider
type Provider struct {
	Type  core.ResourceType `storm:"id"`
	AppID string
	rm    core.ResourceManager
}

// Manager keeps track of all the providers
type Manager struct {
	providers map[core.ResourceType]*Provider
	am        core.AppManager
	db        core.DB
}

// CreateManager returns a Manager, which implements the core.ProviderManager interfaces
func CreateManager(rm core.ResourceManager, am core.AppManager, db core.DB) *Manager {
	providers := map[core.ResourceType]*Provider{}
	providers[core.DNS] = &Provider{Type: core.DNS, rm: rm}
	providers[core.Certificate] = &Provider{Type: core.Certificate, rm: rm}
	providers[core.Mail] = &Provider{Type: core.Mail, rm: rm}

	db.Register(&Provider{})

	prvs := []Provider{}
	err := db.All(&prvs)
	if err != nil {
		log.Fatalf("Could not retrieve providers from the database: %s", err.Error())
	}
	for idx, provider := range prvs {
		prvs[idx].rm = rm
		providers[provider.Type] = &prvs[idx]
	}

	manager := Manager{providers: providers, am: am, db: db}
	return &manager
}

// Register registers a resource provider
func (pm *Manager) Register(app core.App, rtype core.ResourceType) error {
	if pm.providers[rtype].AppID != "" {
		if app.GetID() == pm.providers[rtype].AppID {
			return fmt.Errorf("App %s already registered as a provider for resource type %s", app.GetID(), string(rtype))
		}

		_, err := pm.am.Read(pm.providers[rtype].AppID)
		if err == nil {
			return errors.New("Another application is registered as a provider for resource type " + string(rtype))
		}
	}

	log.Info("Registering provider for resource " + string(rtype))
	pm.providers[rtype].AppID = app.GetID()
	err := pm.db.Save(pm.providers[rtype])
	if err != nil {
		log.Panicf("Failed to save provider to db: %s", err.Error())
	}

	return nil
}

// Deregister deregisters a resource provider
func (pm *Manager) Deregister(app core.App, rtype core.ResourceType) error {

	if pm.providers[rtype].AppID != "" && pm.providers[rtype].AppID != app.GetID() {
		return errors.New("Application '" + app.GetName() + "' is NOT registered for resource type " + string(rtype))
	}

	log.Infof("Deregistering application %s(%s) as a provider for %s", app.GetName(), app.GetID(), string(rtype))
	pm.providers[rtype].AppID = ""
	err := pm.db.Save(pm.providers[rtype])
	if err != nil {
		log.Panicf("Failed to save provider to db: %s", err.Error())
	}
	return nil
}

// Get retrieves the resource provider associated with an app
func (pm *Manager) Get(app core.App) (core.Provider, error) {
	for _, provider := range pm.providers {
		if provider.AppID != "" && provider.AppID == app.GetID() {
			return provider, nil
		}
	}
	return nil, errors.New("Application '" + app.GetName() + "' is NOT a resource provider")
}

//
// Instance methods
//

//GetResources retrieves all resources of a specific resource provider.
func (p *Provider) GetResources() map[string]core.Resource {
	filter := func(rsc core.Resource) bool {
		if rsc.GetType() == p.Type {
			return true
		}
		return false
	}
	rscs := p.rm.Select(filter)
	return rscs
}

//GetResource retrieves a resource that belongs to this provider
func (p *Provider) GetResource(resourceID string) core.Resource {
	rsc, err := p.rm.Get(resourceID)
	if err != nil {
		// ToDo: add custom error reporting or remove the error logging alltogether
		log.Error(err)
		return nil
	}
	if rsc.GetType() != p.Type {
		log.Errorf("Resource %s is not of type %s, but %s", resourceID, p.Type, rsc.GetType())
		return nil
	}

	return rsc
}

//TypeName returns the name of the type of resource the provider provides
func (p *Provider) TypeName() string {
	return string(p.Type)
}
