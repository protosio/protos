package resource

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/protosio/protos/internal/core"

	"github.com/cnf/structhash"
	"github.com/pkg/errors"
)

const (
	resourceDS = "resource"
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
	return nil, fmt.Errorf("Could not find resource '%s'", id)
}

func (rc resourceContainer) remove(id string) error {
	rc.access.Lock()
	defer rc.access.Unlock()
	rsc, found := rc.all[id]
	if found == false {
		return fmt.Errorf("Could not find resource '%s'", id)
	}
	err := rc.db.RemoveFromMap(resourceDS, string(rsc.Type))
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

// CreateManager returns a Manager, which implements the core.ProviderManager interface
func CreateManager(db core.DB) *Manager {
	if db == nil {
		log.Panic("Failed to create  resource manager: none of the inputs can be nil")
	}

	err := db.InitMap(resourceDS, true)
	if err != nil {
		log.Fatal("Failed to initialize resource dataset: ", err)
	}

	log.Debug("Retrieving resources from DB")
	manager := &Manager{db: db}

	rscs := map[string]Resource{}
	err = db.GetMap(resourceDS, &rscs)
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
func (rm *Manager) Create(rtype core.ResourceType, value core.ResourceValue, appID string) (*Resource, error) {
	resource := &Resource{Type: rtype, Value: value}
	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	rsc, err := rm.resources.get(rhash)

	if err == nil {
		return rsc, errors.Wrapf(core.ErrResourceExists{}, "Could not create resource with hash '%s'(%s)", rhash, rtype)
	}

	resource.App = appID
	resource.access = &sync.Mutex{}
	resource.Status = core.Requested
	resource.ID = rhash
	resource.App = appID
	resource.parent = rm
	resource.Save()

	log.Debugf("Adding resource '%s:%+v'", rhash, resource)
	rm.resources.put(rhash, resource)
	return resource, nil

}

//Delete deletes a resource
func (rm *Manager) Delete(appID string) error {
	return rm.resources.remove(appID)
}

//CreateFromJSON creates a resource from the input JSON and adds it to the internal resources map.
func (rm *Manager) CreateFromJSON(appJSON []byte, appID string) (core.Resource, error) {
	rscInitial := &Resource{}
	err := json.Unmarshal(appJSON, rscInitial)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create resource for app '%s'. Error unmarshalling JSON", appID)
	}

	resource := &Resource{Value: rscInitial.Value, Type: rscInitial.Type}
	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	rsc, err := rm.resources.get(rhash)
	if err == nil {
		return rsc, errors.Errorf("Could not create resource with hash '%s' because it already exists", rhash)
	}
	resource.parent = rm
	resource.access = &sync.Mutex{}
	resource.Status = core.Requested
	resource.ID = rhash
	resource.App = appID
	resource.Save()

	log.Debugf("Adding resource '%s:%p'", rhash, resource)
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
func (rm *Manager) GetType(typename string) (core.ResourceType, core.ResourceValue, error) {
	switch typename {
	case "certificate":
		return core.Certificate, &CertificateResource{}, nil
	case "dns":
		return core.DNS, &DNSResource{}, nil
	default:
		return core.ResourceType(""), nil, errors.New("Resource type " + typename + " does not exist")
	}
}

//StringToStatus retrieves a resource status based on the provided string
func (rm *Manager) StringToStatus(statusname string) (core.ResourceStatus, error) {
	switch statusname {
	case "requested":
		return core.Requested, nil
	case "created":
		return core.Created, nil
	case "unknown":
		return core.Unknown, nil
	default:
		return core.ResourceStatus(""), errors.New("Resource status " + statusname + " does not exist")
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

	return rm.Create(core.DNS, val, appID)
}

// CreateCert creates a Resource of type Certificate with the proivded values
func (rm *Manager) CreateCert(appID string, domains []string) (core.Resource, error) {
	val := &CertificateResource{Domains: domains}

	return rm.Create(core.Certificate, val, appID)
}
