package resource

import (
	"encoding/json"
	"sync"

	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
)

var log = util.GetLogger("resource")

// RStatus is a string wrapper used for typechecking the resource status
type RStatus string

const (
	// Requested status is set at creation time and indicates that a resource provider should create this resource
	Requested = RStatus("requested")
	// Created status is the final final state of a resource, ready to be used by an application
	Created = RStatus("created")
	// Unknown status is for error or uknown states
	Unknown = RStatus("unknown")
)

// Resource is the internal abstract representation of things like DNS or TLS certificates.
// Anything that is required for an application to run correctly could and should be modeled as a resource. Think DNS, TLS, IPs, PORTs etc.
type Resource struct {
	access *sync.Mutex

	ID     string  `json:"id" hash:"-"`
	Type   RType   `json:"type"`
	Value  Type    `json:"value"`
	Status RStatus `json:"status"`
}

//
// Resource
//

// Save - persists application data to database
func (rsc *Resource) Save() {
	rsc.access.Lock()
	err := database.Save(rsc)
	rsc.access.Unlock()
	if err != nil {
		log.Panicf("Failed to save resource to db: %s", err.Error())
	}
}

//Delete deletes a resource
func (rsc *Resource) Delete() error {
	log.Debug("Deleting resource " + rsc.ID)
	return resources.remove(rsc.ID)
}

// SetStatus sets the status on a resource instance
func (rsc *Resource) SetStatus(status RStatus) {
	rsc.access.Lock()
	rsc.Status = status
	rsc.access.Unlock()
	rsc.Save()
}

// UpdateValue updates the value of a resource
func (rsc *Resource) UpdateValue(value Type) {
	rsc.access.Lock()
	rsc.Value.Update(value)
	rsc.access.Unlock()
	rsc.Save()
}

// Sanitize returns a sanitized version of the resource, with sensitive fields removed
func (rsc *Resource) Sanitize() Resource {
	rsc.access.Lock()
	srsc := *rsc
	rsc.access.Unlock()
	srsc.Value = rsc.Value.Sanitize()
	return srsc
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
