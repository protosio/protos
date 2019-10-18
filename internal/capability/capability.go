package capability

import (
	"encoding/gob"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"

	"github.com/pkg/errors"
)

var log = util.GetLogger("capability")

// AllCapabilities is a list of all the capabilities available in the system
var AllCapabilities = []*Capability{}

//CapMap holds a maping of methods to capabilities
var CapMap = make(map[string]*Capability)

// RC is the root capability
var RC *Capability

// Checker is an interface that implements methods for checking a capability
type Checker interface {
	ValidateCapability(cap *Capability) error
}

// Capability represents a security capability in the system
type Capability struct {
	Name   string `storm:"id"`
	Parent *Capability
}

// Manager implements the core.CapabilityManager interface
type Manager struct {
	root *Capability
}

// CreateManager creates the root capability and returns a core.CapabilityManager
func CreateManager() *Manager {
	log.Info("Initializing capabilities")
	cm := &Manager{}
	cm.root = cm.New("RootCapability")
	cm.createTree(cm.root)
	gob.Register(&Capability{})

	return cm
}

// New returns a new capability
func (cm *Manager) New(name string) *Capability {
	log.Debugf("Creating capability '%s'", name)
	cap := &Capability{Name: name}
	AllCapabilities = append(AllCapabilities, cap)
	return cap
}

// Validate validates a capability
func (cm *Manager) Validate(methodcap core.Capability, appcap string) bool {
	if methodcap.GetName() == appcap {
		log.Debugf("Matched capability at '%s'", methodcap.GetName())
		return true
	} else if methodcap.GetParent() != nil {
		return cm.Validate(methodcap.GetParent(), appcap)
	}
	return false
}

// SetMethodCap adds a capability for a specific method
func (cm *Manager) SetMethodCap(method string, cap core.Capability) {
	lcap := cap.(*Capability)
	log.Debugf("Setting capability '%s' for method '%s'", lcap.Name, method)
	CapMap[method] = lcap
}

// GetMethodCap returns a capability for a specific method
func (cm *Manager) GetMethodCap(method string) (core.Capability, error) {
	if cap, ok := CapMap[method]; ok {
		return cap, nil
	}
	return nil, errors.Errorf("Can't find capability for method '%s'", method)
}

// GetByName returns the capability based on the provided name, if one exists
func (cm *Manager) GetByName(name string) (core.Capability, error) {
	for _, cap := range AllCapabilities {
		if cap.Name == name {
			return cap, nil
		}
	}
	return nil, errors.Errorf("Capability '%s' does not exist", name)
}

// GetOrPanic returns the capability based on the provided name. It panics if it's not found
func (cm *Manager) GetOrPanic(name string) core.Capability {
	for _, cap := range AllCapabilities {
		if cap.Name == name {
			return cap
		}
	}
	log.Panicf("Could not find capability '%s'", name)
	return nil
}

//
// Capability methods that implement the core.Capability interface
//

// SetParent takes a capability and sets it as the parent
func (cap *Capability) SetParent(parent *Capability) {
	cap.Parent = parent
}

// GetName returns the name of the capability
func (cap *Capability) GetName() string {
	return cap.Name
}

// GetParent returns the parent of the capability
func (cap *Capability) GetParent() core.Capability {
	if cap.Parent == nil {
		return nil
	}
	return cap.Parent
}
