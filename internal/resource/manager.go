package resource

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/protosio/protos/internal/db"

	"github.com/cnf/structhash"
	"github.com/pkg/errors"
)

const (
	resourceDS = "resource"
)

// Manager keeps track of all the resources
type Manager struct {
	db db.DB
}

//
// Public methods that satisfy *ResourceManager
//

// CreateManager returns a Manager, which implements the core.ProviderManager interface
func CreateManager(db db.DB) *Manager {
	if db == nil {
		log.Panic("Failed to create  resource manager: none of the inputs can be nil")
	}

	err := db.InitDataset(resourceDS, true)
	if err != nil {
		log.Fatal("Failed to initialize resource dataset: ", err)
	}

	log.Debug("Retrieving resources from DB")
	manager := &Manager{db: db}
	return manager
}

//Create creates a resource and adds it to the internal resources map.
func (rm *Manager) Create(rtype ResourceType, value ResourceValue, appID string) (*Resource, error) {
	resource := &Resource{Type: rtype, Value: value}
	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))

	_, err := rm.Get(rhash)
	if err == nil {
		return resource, errors.Wrapf(ErrResourceExists{}, "Could not create resource with hash '%s'(%s) because it already exists", rhash, rtype)
	}

	resource.App = appID
	resource.access = &sync.Mutex{}
	resource.Status = Requested
	resource.ID = rhash
	resource.App = appID
	resource.parent = rm
	resource.Save()

	return resource, nil

}

//Delete deletes a resource
func (rm *Manager) Delete(id string) error {
	err := rm.db.RemoveFromMap(resourceDS, id)
	if err != nil {
		return fmt.Errorf("Could not remove resource from database: %s", err.Error())
	}

	return nil
}

//CreateFromJSON creates a resource from the input JSON and adds it to the internal resources map.
func (rm *Manager) CreateFromJSON(appJSON []byte, appID string) (*Resource, error) {
	rscInitial := &Resource{}
	err := json.Unmarshal(appJSON, rscInitial)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create resource for app '%s'. Error unmarshalling JSON", appID)
	}

	resource := &Resource{Value: rscInitial.Value, Type: rscInitial.Type}
	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	_, err = rm.Get(rhash)
	if err == nil {
		return resource, errors.Wrapf(ErrResourceExists{}, "Could not create resource with hash '%s'(%s) because it already exists", rhash, rscInitial.Type)
	}
	resource.parent = rm
	resource.access = &sync.Mutex{}
	resource.Status = Requested
	resource.ID = rhash
	resource.App = appID
	resource.Save()

	return resource, nil
}

// Select takes a function and applies it to all the resources in the map. The ones that return true are returned
func (rm *Manager) Select(filter func(*Resource) bool) map[string]*Resource {
	selectedResources := map[string]*Resource{}

	rscs := map[string]Resource{}
	err := rm.db.GetMap(resourceDS, &rscs)
	if err != nil {
		log.Fatalf("Could not retrieve resources from the database: %s", err.Error())
	}

	for k, v := range rscs {
		rsc := &v
		if filter(rsc) {
			selectedResources[k] = rsc
		}
	}

	return selectedResources
}

//GetAll retrieves all the saved resources
func (rm *Manager) GetAll(sanitize bool) map[string]*Resource {

	rscs := map[string]Resource{}
	err := rm.db.GetMap(resourceDS, &rscs)
	if err != nil {
		log.Fatalf("Could not retrieve resources from the database: %s", err.Error())
	}

	var sanitizedResources = make(map[string]*Resource, len(rscs))
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
func (rm *Manager) Get(id string) (*Resource, error) {

	rscs := map[string]Resource{}
	err := rm.db.GetMap(resourceDS, &rscs)
	if err != nil {
		log.Fatalf("Could not retrieve resources from the database: %s", err.Error())
	}

	rsc, found := rscs[id]
	if found {
		return &rsc, nil
	}

	return nil, fmt.Errorf("Could not find resource '%s'", id)
}

//GetType retrieves a resource type based on the provided string
func (rm *Manager) GetType(typename string) (ResourceType, ResourceValue, error) {
	switch typename {
	case "certificate":
		return Certificate, &CertificateResource{}, nil
	case "dns":
		return DNS, &DNSResource{}, nil
	default:
		return ResourceType(""), nil, errors.New("Resource type " + typename + " does not exist")
	}
}

//StringToStatus retrieves a resource status based on the provided string
func (rm *Manager) StringToStatus(statusname string) (ResourceStatus, error) {
	switch statusname {
	case "requested":
		return Requested, nil
	case "created":
		return Created, nil
	case "unknown":
		return Unknown, nil
	default:
		return ResourceStatus(""), errors.New("Resource status " + statusname + " does not exist")
	}
}

//
// Public methods that satisfy *ResourceCreator
//

// CreateDNS creates a Resource of type DNS with the proivded values
func (rm *Manager) CreateDNS(appID string, name string, rtype string, value string, ttl int) (*Resource, error) {
	val := &DNSResource{
		Host:  name,
		Type:  rtype,
		Value: value,
		TTL:   ttl,
	}

	return rm.Create(DNS, val, appID)
}

// CreateCert creates a Resource of type Certificate with the proivded values
func (rm *Manager) CreateCert(appID string, domains []string) (*Resource, error) {
	val := &CertificateResource{Domains: domains}

	return rm.Create(Certificate, val, appID)
}
