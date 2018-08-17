package resource

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/util"

	"github.com/cnf/structhash"
)

var log = util.Log

type RStatus string

const (
	Requested = RStatus("requested")
	Created   = RStatus("created")
	Unknown   = RStatus("unknown")
)

type Resource struct {
	ID     string  `json:"id" hash:"-"`
	Type   RType   `json:"type"`
	Value  Type    `json:"value"`
	Status RStatus `json:"status"`
}

var resources = make(map[string]*Resource)

//
// Resource
//

//GetAll retrieves all the saved resources
func GetAll(sanitize bool) map[string]*Resource {
	if sanitize == false {
		return resources
	}
	var sanitizedResources = make(map[string]*Resource, len(resources))
	for _, rsc := range resources {
		sanitizedResources[rsc.ID] = rsc.Sanitize()
	}
	return sanitizedResources
}

//GetForType returns all the resources of a specific resource type
func GetForType(RType RType) []*Resource {
	rscs := []*Resource{}
	for _, rsc := range resources {
		if rsc.Type == RType {
			rscs = append(rscs, rsc)
		}
	}
	return rscs
}

//CreateFromJSON creates a resource from the input JSON and adds it to the internal resources map.
func CreateFromJSON(appJSON []byte) (*Resource, error) {

	resource := &Resource{}
	err := json.Unmarshal(appJSON, resource)
	if err != nil {
		return resource, err
	}

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	if _, ok := resources[rhash]; ok {
		return &Resource{}, errors.New("Resource " + rhash + " already registered")
	}
	resource.Status = Requested
	resource.ID = rhash

	resource.Save()

	log.Debug("Adding resource ", rhash, ": ", resource)
	resources[rhash] = resource
	return resource, nil
}

//Create creates a resource and adds it to the internal resources map.
func Create(rtype RType, value Type) (*Resource, error) {
	resource := &Resource{}
	resource.Type = rtype
	resource.Value = value

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	if rsc, ok := resources[rhash]; ok {
		return rsc, errors.New("Resource " + rhash + " already registered")
	}
	resource.Status = Requested
	resource.ID = rhash

	resource.Save()

	log.Debug("Adding resource ", rhash, ": ", resource)
	resources[rhash] = resource
	return resource, nil

}

//Get retrieves a resources based on the provided id
func Get(resourceID string) (*Resource, error) {
	rsc, ok := resources[resourceID]
	if ok != true {
		return nil, errors.New("Resource " + resourceID + " does not exist.")
	}
	return rsc, nil
}

// Save - persists application data to database
func (rsc *Resource) Save() {
	err := database.Save(rsc)
	if err != nil {
		log.Panicf("Failed to save resource to db: %s", err.Error())
	}
}

//Delete deletes a resource
func (rsc *Resource) Delete() error {
	_, ok := resources[rsc.ID]
	if ok != true {
		return errors.New("Resource " + rsc.ID + " does not exist.")
	}

	log.Debug("Deleting resource " + rsc.ID)
	err := database.Remove(rsc)
	if err != nil {
		log.Panicf("Failed to remove resource from db: %s", err.Error())
	}
	delete(resources, rsc.ID)
	return nil
}

// SetStatus sets the status on a resource instance
func (rsc *Resource) SetStatus(status RStatus) {
	rsc.Status = status
	rsc.Save()
}

// UpdateValue updates the value of a resource
func (rsc *Resource) UpdateValue(value Type) {
	rsc.Value.Update(value)
	rsc.Save()
}

// Sanitize returns a sanitized version of the resource, with sensitive fields removed
func (rsc *Resource) Sanitize() *Resource {
	srsc := *rsc
	srsc.Value = rsc.Value.Sanitize()
	return &srsc
}

// UnmarshalJSON is a custom json unmarshaller for resource
func (rsc *Resource) UnmarshalJSON(b []byte) error {
	resdata := struct {
		ID     string          `json:"id" hash:"-"`
		Type   RType           `json:"type"`
		Value  json.RawMessage `json:"value"`
		Status RStatus         `json:"status"`
	}{}
	err := json.Unmarshal(b, &resdata)
	if err != nil {
		return err
	}

	rsc.ID = resdata.ID
	rsc.Type = resdata.Type
	rsc.Status = resdata.Status
	_, resourceStruct, err := GetType(string(resdata.Type))
	if err != nil {
		return err
	}

	err = json.Unmarshal(resdata.Value, &resourceStruct)
	if err != nil {
		return err
	}
	rsc.Value = resourceStruct
	return nil
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

// LoadResourcesDB loads resources from the database
func LoadResourcesDB() {
	log.Info("Retrieving resources from DB")
	gob.Register(&Resource{})
	gob.Register(&DNSResource{})
	gob.Register(&CertificateResource{})

	rscs := []Resource{}
	err := database.All(&rscs)
	if err != nil {
		log.Fatalf("Could not retrieve resources from the database: %s", err.Error())
	}
	for idx, rsc := range rscs {
		resources[rsc.ID] = &rscs[idx]
	}
}
