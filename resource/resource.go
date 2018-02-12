package resource

import (
	"encoding/json"
	"errors"
	"fmt"

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
func GetAll() map[string]*Resource {
	return resources
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

//Create creates a resource and adds it to the internal resources map.
func Create(appJSON []byte) (*Resource, error) {

	resource := &Resource{}
	err := json.Unmarshal(appJSON, resource)
	if err != nil {
		return resource, err
	}

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	if _, ok := resources[rhash]; ok {
		return &Resource{}, errors.New("Resource " + rhash + " already registered")
	}
	log.Debug("Adding resource ", rhash, ": ", resource)

	resource.Status = Requested
	resource.ID = rhash
	resources[rhash] = resource
	return resource, nil
}

//Delete deletes a resource
func Delete(resourceID string) error {
	_, ok := resources[resourceID]
	if ok != true {
		return errors.New("Resource " + resourceID + " does not exist.")
	}

	log.Info("Deleting resource " + resourceID)
	delete(resources, resourceID)
	return nil
}

//Get retrieves a resources based on the provided id
func Get(resourceID string) (*Resource, error) {
	rsc, ok := resources[resourceID]
	if ok != true {
		return nil, errors.New("Resource " + resourceID + " does not exist.")
	}
	return rsc, nil
}

// SetStatus allows a provider to modify the status of a resource
func SetStatus(resourceID string, status RStatus) error {
	resource, ok := resources[resourceID]
	if ok != true {
		return errors.New("Resource [" + resourceID + "] does not exist.")
	}

	log.Debug("Setting status ", status, " for resource ", resourceID)
	resource.Status = status

	return nil

}

// SetStatus sets the status on a resource instance
func (rsc *Resource) SetStatus(status RStatus) {
	rsc.Status = status
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
