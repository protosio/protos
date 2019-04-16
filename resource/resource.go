package resource

import (
	"encoding/json"
	"sync"

	"github.com/protosio/protos/core"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
)

var log = util.GetLogger("resource")

// Resource is the internal abstract representation of things like DNS or TLS certificates.
// Anything that is required for an application to run correctly could and should be modeled as a resource. Think DNS, TLS, IPs, PORTs etc.
type Resource struct {
	access *sync.Mutex
	parent *Manager

	ID     string       `json:"id" hash:"-"`
	Type   core.RType   `json:"type"`
	Value  core.Type    `json:"value"`
	Status core.RStatus `json:"status"`
	App    string       `json:"app"`
}

//
// Resource
//

// GetID returns the string ID of the resource
func (rsc *Resource) GetID() string {
	return rsc.ID
}

// GetAppID returns the ID of the parent application
func (rsc *Resource) GetAppID() string {
	return rsc.App
}

// Save persists application data to database
func (rsc *Resource) Save() {
	rsc.access.Lock()
	err := database.Save(rsc)
	rsc.access.Unlock()
	if err != nil {
		log.Panicf("Failed to save resource to db: %s", err.Error())
	}
}

// SetStatus sets the status on a resource instance
func (rsc *Resource) SetStatus(status core.RStatus) {
	rsc.access.Lock()
	rsc.Status = status
	rsc.access.Unlock()
	rsc.Save()
}

// UpdateValue updates the value of a resource
func (rsc *Resource) UpdateValue(value core.Type) {
	rsc.access.Lock()
	rsc.Value.Update(value)
	rsc.access.Unlock()
	rsc.Save()
}

// GetType returns the type of the resources
func (rsc *Resource) GetType() core.RType {
	return rsc.Type
}

// GetValue returns the encapsulated value of the resource
func (rsc *Resource) GetValue() core.Type {
	return rsc.Value
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
		Type   core.RType      `json:"type"`
		Value  json.RawMessage `json:"value"`
		Status core.RStatus    `json:"status"`
	}{}
	err := json.Unmarshal(b, &resdata)
	if err != nil {
		return err
	}

	rsc.ID = resdata.ID
	rsc.Type = resdata.Type
	rsc.Status = resdata.Status
	_, resourceStruct, err := rsc.parent.GetType(string(resdata.Type))
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
