package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"protos/internal/core"

	"github.com/cnf/structhash"
)

// resourceContainer is a thread safe application map
type resourceContainer struct {
	access *sync.Mutex
	all    map[string]*Resource
	db     core.DB
}

// put saves an application into the application map
func (rc resourceContainer) put(id string, app *Resource) {
	rc.access.Lock()
	rc.all[id] = app
	rc.access.Unlock()
}

// get retrieves an application from the application map
func (rc resourceContainer) get(id string) (*Resource, error) {
	rc.access.Lock()
	app, found := rc.all[id]
	rc.access.Unlock()
	if found {
		return app, nil
	}
	return nil, fmt.Errorf("Could not find app %s", id)
}

func (rc resourceContainer) remove(id string) error {
	rc.access.Lock()
	defer rc.access.Unlock()
	rsc, found := rc.all[id]
	if found == false {
		return fmt.Errorf("Could not find app %s", id)
	}
	err := rc.db.Remove(rsc)
	if err != nil {
		log.Panicf("Failed to remove resource from db: %s", err.Error())
	}
	delete(rc.all, id)
	return nil
}

// copy returns a copy of the applications map
func (rc resourceContainer) copy() map[string]Resource {
	copyResources := map[string]Resource{}
	rc.access.Lock()
	defer rc.access.Unlock()
	for k, v := range rc.all {
		v.access.Lock()
		rsc := *v
		v.access.Unlock()
		copyResources[k] = rsc
	}
	return copyResources
}

// Manager keeps track of all the resources
type Manager struct {
	resources resourceContainer
	db        core.DB
}

//
// Public methods that satisfy core.ResourceManager
//

// CreateManager returns a Manager, which implements the core.ProviderManager interfaces
func CreateManager(db core.DB) *Manager {
	log.Debug("Retrieving resources from DB")
	db.Register(&Resource{})
	db.Register(&DNSResource{})
	db.Register(&CertificateResource{})
	manager := &Manager{db: db}

	rscs := []Resource{}
	err := db.All(&rscs)
	if err != nil {
		log.Fatalf("Could not retrieve resources from the database: %s", err.Error())
	}
	resources := resourceContainer{access: &sync.Mutex{}, all: map[string]*Resource{}, db: db}
	for _, rsc := range rscs {
		rscCopy := rsc
		rscCopy.access = &sync.Mutex{}
		rscCopy.parent = manager
		resources.put(rsc.ID, &rscCopy)
	}

	manager.resources = resources
	return manager
}

//Create creates a resource and adds it to the internal resources map.
func (rm *Manager) Create(rtype core.RType, value core.Type, appID string) (*Resource, error) {
	resource := &Resource{access: &sync.Mutex{}, App: appID}
	resource.Type = rtype
	resource.Value = value

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	rsc, err := rm.resources.get(rhash)
	if err == nil {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource.Status = core.Requested
	resource.ID = rhash
	resource.App = appID
	resource.Save()

	log.Debug("Adding resource ", rhash, ": ", resource)
	rm.resources.put(rhash, resource)
	return resource, nil

}

//Delete deletes a resource
func (rm *Manager) Delete(appID string) error {
	return rm.resources.remove(appID)
}

//CreateFromJSON creates a resource from the input JSON and adds it to the internal resources map.
func (rm *Manager) CreateFromJSON(appJSON []byte, appID string) (core.Resource, error) {
	resource := &Resource{access: &sync.Mutex{}}
	err := json.Unmarshal(appJSON, resource)
	if err != nil {
		return resource, err
	}

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	rsc, err := rm.resources.get(rhash)
	if err == nil {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource.Status = core.Requested
	resource.ID = rhash
	resource.App = appID
	resource.Save()

	log.Debug("Adding resource ", rhash, ": ", resource)
	rm.resources.put(rhash, resource)
	return resource, nil
}

// Select takes a function and applies it to all the resources in the map. The ones that return true are returned
func (rm *Manager) Select(filter func(core.Resource) bool) map[string]core.Resource {
	selectedResources := map[string]core.Resource{}
	rm.resources.access.Lock()
	for k, v := range rm.resources.all {
		rsc := v
		rsc.access.Lock()
		if filter(rsc) {
			selectedResources[k] = rsc
		}
		rsc.access.Unlock()
	}
	rm.resources.access.Unlock()
	return selectedResources
}

//GetAll retrieves all the saved resources
func (rm *Manager) GetAll(sanitize bool) map[string]core.Resource {
	rscs := rm.resources.copy()
	var sanitizedResources = make(map[string]core.Resource, len(rscs))
	for _, rsc := range rscs {
		if sanitize == false {
			sanitizedResources[rsc.ID] = &rsc
		} else {
			sanitizedResources[rsc.ID] = rsc.Sanitize()
		}
	}
	return sanitizedResources
}

//Get retrieves a resources based on the provided id
func (rm *Manager) Get(resourceID string) (core.Resource, error) {
	return rm.resources.get(resourceID)
}

//GetType retrieves a resource type based on the provided string
func (rm *Manager) GetType(typename string) (core.RType, core.Type, error) {
	switch typename {
	case "certificate":
		return Certificate, &CertificateResource{}, nil
	case "dns":
		return DNS, &DNSResource{}, nil
	default:
		return core.RType(""), nil, errors.New("Resource type " + typename + " does not exist")
	}
}

//GetStatus retrieves a resource status based on the provided string
func (rm *Manager) GetStatus(statusname string) (core.RStatus, error) {
	switch statusname {
	case "requested":
		return core.Requested, nil
	case "created":
		return core.Created, nil
	case "unknown":
		return core.Unknown, nil
	default:
		return core.RStatus(""), errors.New("Resource status " + statusname + " does not exist")
	}
}

//
// Public methods that satisfy core.ResourceCreator
//

// CreateDNS creates a Resource of type DNS with the proivded values
func (rm *Manager) CreateDNS(appID string, name string, rtype string, value string, ttl int) (core.Resource, error) {
	val := &DNSResource{
		Host:  name,
		Type:  rtype,
		Value: value,
		TTL:   ttl,
	}
	rhash := fmt.Sprintf("%x", structhash.Md5(val, 1))
	rsc, err := rm.resources.get(rhash)
	if err == nil {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource := &Resource{access: &sync.Mutex{}, parent: rm, ID: rhash, App: appID, Type: DNS, Value: val, Status: core.Requested}
	resource.Save()

	return resource, nil
}

// CreateCert creates a Resource of type Certificate with the proivded values
func (rm *Manager) CreateCert(appID string, domains []string) (core.Resource, error) {
	val := &CertificateResource{Domains: domains}
	rhash := fmt.Sprintf("%x", structhash.Md5(val, 1))
	rsc, err := rm.resources.get(rhash)
	if err == nil {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource := &Resource{access: &sync.Mutex{}, parent: rm, ID: rhash, App: appID, Type: Certificate, Value: val, Status: core.Requested}
	resource.Save()

	return resource, nil
}
