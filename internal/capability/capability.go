package capability

import (
	"encoding/gob"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"

	"github.com/pkg/errors"
)

var log = util.GetLogger("capability")

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
	root            *Capability
	capMap          map[string]*Capability
	allCapabilities []*Capability
}

// CreateManager creates the root capability and returns a core.CapabilityManager
func CreateManager() *Manager {
	log.Info("Initializing capabilities")
	cm := &Manager{}
	cm.root = cm.New("RootCapability")
	cm.capMap = make(map[string]*Capability)
	cm.createTree(cm.root)
	gob.Register(&Capability{})

	return cm
}

// New returns a new capability
func (cm *Manager) New(name string) *Capability {
	log.Tracef("Creating capability '%s'", name)
	cap := &Capability{Name: name}
	cm.allCapabilities = append(cm.allCapabilities, cap)
	return cap
}

// Validate validates a capability
func (cm *Manager) Validate(methodcap core.Capability, appcap string) bool {
	if methodcap.GetName() == appcap {
		log.Tracef("Matched capability at '%s'", methodcap.GetName())
		return true
	} else if methodcap.GetParent() != nil {
		return cm.Validate(methodcap.GetParent(), appcap)
	}
	return false
}

// SetMethodCap adds a capability for a specific method
func (cm *Manager) SetMethodCap(method string, cap core.Capability) {
	lcap := cap.(*Capability)
	log.Tracef("Setting capability '%s' for method '%s'", lcap.Name, method)
	cm.capMap[method] = lcap
}

// GetMethodCap returns a capability for a specific method
func (cm *Manager) GetMethodCap(method string) (core.Capability, error) {
	if cap, ok := cm.capMap[method]; ok {
		return cap, nil
	}
	return nil, errors.Errorf("Can't find capability for method '%s'", method)
}

// GetByName returns the capability based on the provided name, if one exists
func (cm *Manager) GetByName(name string) (core.Capability, error) {
	for _, cap := range cm.allCapabilities {
		if cap.Name == name {
			return cap, nil
		}
	}
	return nil, errors.Errorf("Capability '%s' does not exist", name)
}

// GetOrPanic returns the capability based on the provided name. It panics if it's not found
func (cm *Manager) GetOrPanic(name string) core.Capability {
	for _, cap := range cm.allCapabilities {
		if cap.Name == name {
			return cap
		}
	}
	log.Panicf("Could not find capability '%s'", name)
	return nil
}

// ClearAll removes all the association between methods and capabilities
func (cm *Manager) ClearAll() {
	cm.capMap = make(map[string]*Capability)
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
