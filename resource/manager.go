package resource

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/cnf/structhash"
	"github.com/protosio/protos/database"
)

// resourceContainer is a thread safe application map
type resourceContainer struct {
	access *sync.Mutex
	all    map[string]*Resource
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
	err := database.Remove(rsc)
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

var resources resourceContainer

//
// Public methods
//

//Create creates a resource and adds it to the internal resources map.
func Create(rtype RType, value Type) (*Resource, error) {
	resource := &Resource{}
	resource.Type = rtype
	resource.Value = value

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	rsc, err := resources.get(rhash)
	if err == nil {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource.Status = Requested
	resource.ID = rhash
	resource.Save()

	log.Debug("Adding resource ", rhash, ": ", resource)
	resources.put(rhash, resource)
	return resource, nil

}

//CreateFromJSON creates a resource from the input JSON and adds it to the internal resources map.
func CreateFromJSON(appJSON []byte) (*Resource, error) {

	resource := &Resource{}
	err := json.Unmarshal(appJSON, resource)
	if err != nil {
		return resource, err
	}

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	rsc, err := resources.get(rhash)
	if err == nil {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource.Status = Requested
	resource.ID = rhash
	resource.Save()

	log.Debug("Adding resource ", rhash, ": ", resource)
	resources.put(rhash, resource)
	return resource, nil
}

// Select takes a function and applies it to all the resources in the map. The ones that return true are returned
func Select(filter func(*Resource) bool) map[string]*Resource {
	selectedResources := map[string]*Resource{}
	resources.access.Lock()
	for k, v := range resources.all {
		rsc := v
		rsc.access.Lock()
		if filter(rsc) {
			selectedResources[k] = rsc
		}
		rsc.access.Unlock()
	}
	resources.access.Unlock()
	return selectedResources
}

//GetAll retrieves all the saved resources
func GetAll(sanitize bool) map[string]Resource {
	rscs := resources.copy()
	if sanitize == false {
		return rscs
	}
	var sanitizedResources = make(map[string]Resource, len(rscs))
	for _, rsc := range rscs {
		sanitizedResources[rsc.ID] = rsc.Sanitize()
	}
	return sanitizedResources
}

//Get retrieves a resources based on the provided id
func Get(resourceID string) (*Resource, error) {
	return resources.get(resourceID)
}

//GetType retrieves a resource type based on the provided string
func GetType(typename string) (RType, Type, error) {
	switch typename {
	case "certificate":
		return Certificate, &CertificateResource{}, nil
	case "dns":
		return DNS, &DNSResource{}, nil
	default:
		return RType(""), nil, errors.New("Resource type " + typename + " does not exist")
	}
}

//GetStatus retrieves a resource status based on the provided string
func GetStatus(statusname string) (RStatus, error) {
	switch statusname {
	case "requested":
		return Requested, nil
	case "created":
		return Created, nil
	case "unknown":
		return Unknown, nil
	default:
		return RStatus(""), errors.New("Resource status " + statusname + " does not exist")
	}
}

// Init loads resources from the database
func Init() {
	log.Info("Retrieving resources from DB")
	gob.Register(&Resource{})
	gob.Register(&DNSResource{})
	gob.Register(&CertificateResource{})

	rscs := []Resource{}
	err := database.All(&rscs)
	if err != nil {
		log.Fatalf("Could not retrieve resources from the database: %s", err.Error())
	}
	resources = resourceContainer{access: &sync.Mutex{}, all: map[string]*Resource{}}
	for _, rsc := range rscs {
		rscCopy := rsc
		rscCopy.access = &sync.Mutex{}
		resources.put(rsc.ID, &rscCopy)
	}
}
