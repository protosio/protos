package resource

import (
	"encoding/json"
	"errors"
	"fmt"
	"protos/util"

	"github.com/cnf/structhash"
	"github.com/tidwall/gjson"
)

var log = util.Log

type RStatus string
type RType string

const (
	Requested = RStatus("requested")
	Created   = RStatus("created")
	Unknown   = RStatus("unknown")
)

type Resource struct {
	ID     string      `json:"id" hash:"-"`
	Type   RType       `json:"type"`
	Fields interface{} `json:"value"`
	Status RStatus     `json:"status"`
}

var resources = make(map[string]*Resource)

//
// Resource
//

//GetAll retrieves all the saved resources
// some fields are modified before being returned
func GetAll() map[string]interface{} {
	modifiedResources := make(map[string]interface{})
	for id, rsc := range resources {
		mrsc := struct {
			ID     string      `json:"id" hash:"-"`
			Type   RType       `json:"type"`
			Fields interface{} `json:"value"`
			Status RStatus     `json:"status"`
		}{
			ID:     rsc.ID,
			Type:   rsc.Type,
			Fields: rsc.Fields,
			Status: rsc.Status,
		}
		modifiedResources[id] = mrsc
	}
	return modifiedResources
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

//GetResourceFromJSON recevies json and casts it to the correct data structure
func GetResourceFromJSON(resourceJSON []byte) (*Resource, error) {

	resource := Resource{}
	err := json.Unmarshal(resourceJSON, &resource)
	if err != nil {
		return &Resource{}, err
	}

	resourceJSONValue := gjson.GetBytes(resourceJSON, "value")
	var raw []byte
	if resourceJSONValue.Index > 0 {
		raw = resourceJSON[resourceJSONValue.Index : resourceJSONValue.Index+len(resourceJSONValue.Raw)]
	} else {
		raw = []byte(resourceJSONValue.Raw)
	}

	resourceType := gjson.Get(string(resourceJSON), "type").Str
	if resourceType == string(DNS) {
		resourceStruct := DNSResource{}
		err = json.Unmarshal(raw, &resourceStruct)
		if err != nil {
			return &Resource{}, err
		}
		resource.Fields = resourceStruct
	} else {
		return &Resource{}, errors.New("Resource type '" + resourceType + "' does not exists")
	}

	return &resource, nil
}

//Create creates a resource and adds it to the internal resources map.
func Create(appJSON []byte) (*Resource, error) {

	resource, err := GetResourceFromJSON(appJSON)
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
func GetType(typename string) (RType, error) {
	switch typename {
	case "certificate":
		return Certificate, nil
	case "dns":
		return DNS, nil
	case "mail":
		return Mail, nil
	default:
		return RType(""), errors.New("Resource type " + typename + " does not exist")
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
