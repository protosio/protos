package resource

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("resource")

// ErrResourceExists is returned when trying to create a resource that already exists
type ErrResourceExists struct{}

func (e ErrResourceExists) Error() string {
	return "Resource exists"
}

// ResourceValue is an interface that satisfies all the resource values
type ResourceValue interface {
	Update(ResourceValue)
	Sanitize() ResourceValue
}

// ResourceStatus is a string wrapper used for typechecking the resource status
type ResourceStatus string

const (
	// Requested status is set at creation time and indicates that a resource provider should create this resource
	Requested = ResourceStatus("requested")
	// Created status is the final final state of a resource, ready to be used by an application
	Created = ResourceStatus("created")
	// Unknown status is for error or uknown states
	Unknown = ResourceStatus("unknown")
)

// Resource is the internal abstract representation of things like DNS or TLS certificates.
// Anything that is required for an application to run correctly could and should be modeled as a resource. Think DNS, TLS, IPs, PORTs etc.
type Resource struct {
	access *sync.Mutex `noms:"-"`
	parent *Manager    `noms:"-"`

	// Public members
	ID     string         `json:"id" hash:"-"`
	Type   ResourceType   `json:"type"`
	Value  ResourceValue  `json:"value" noms:"omitempty"`
	Status ResourceStatus `json:"status"`
	App    string         `json:"app"`
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
	err := rsc.parent.db.InsertInMap(resourceDS, rsc.ID, *rsc)
	rsc.access.Unlock()
	if err != nil {
		log.Panicf("Failed to save resource to db: %s", err.Error())
	}
}

// GetStatus sets the status on a resource instance
func (rsc *Resource) GetStatus() ResourceStatus {
	return rsc.Status
}

// SetStatus sets the status on a resource instance
func (rsc *Resource) SetStatus(status ResourceStatus) {
	rsc.access.Lock()
	rsc.Status = status
	rsc.access.Unlock()
	rsc.Save()
}

// UpdateValue updates the value of a resource
func (rsc *Resource) UpdateValue(value ResourceValue) {
	log.Debugf("Updating resource '%s' of type '%s': %v+", rsc.ID, rsc.Type, value)
	rsc.access.Lock()
	rsc.Value.Update(value)
	rsc.access.Unlock()
	rsc.Save()
}

// GetType returns the type of the resources
func (rsc *Resource) GetType() ResourceType {
	return rsc.Type
}

// GetValue returns the encapsulated value of the resource
func (rsc *Resource) GetValue() ResourceValue {
	return rsc.Value
}

// Sanitize returns a sanitized version of the resource, with sensitive fields removed
func (rsc *Resource) Sanitize() *Resource {
	rsc.access.Lock()
	srsc := *rsc
	rsc.access.Unlock()
	srsc.Value = srsc.Value.Sanitize()
	return &srsc
}

// UnmarshalJSON is a custom json unmarshaller for resource
func (rsc *Resource) UnmarshalJSON(b []byte) error {
	resdata := struct {
		ID     string          `json:"id" hash:"-"`
		Type   ResourceType    `json:"type"`
		Value  json.RawMessage `json:"value"`
		Status ResourceStatus  `json:"status"`
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

// UnmarshalNoms decodes the resource value from a noms value type
func (rsc *Resource) UnmarshalNoms(v types.Value) error {
	values := []types.Value{}
	v.WalkValues(func(v types.Value) {
		values = append(values, v)
	})

	if len(values) == 5 && values[4].Kind() == types.StringKind {
		var rscv ResourceValue
		var err error

		switch string(values[4].(types.String)) {
		case "certificate":
			rscs := CertificateResource{}
			err = marshal.Unmarshal(values[2], &rscs)
			rscv = &rscs
		case "dns":
			rscs := DNSResource{}
			err = marshal.Unmarshal(values[2], &rscs)
			rscv = &rscs
		default:
			return fmt.Errorf("Resource type '%s' does not exist", string(values[4].(types.String)))
		}
		if err != nil {
			return fmt.Errorf("Failed tu unmarshall Resource from Noms: %w", err)
		}
		rsc.Value = rscv
	} else {
		return fmt.Errorf("Failed tu unmarshall Resource from Noms: wrong number of elements in struct or wrong type")
	}

	rsc.App = string(values[0].(types.String))
	rsc.ID = string(values[1].(types.String))
	rsc.Status = ResourceStatus(values[3].(types.String))
	rsc.Type = ResourceType(values[4].(types.String))

	return nil
}
