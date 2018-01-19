package capability

import (
	"errors"
	"protos/util"
	"reflect"
	"runtime"

	"github.com/rs/xid"
)

var log = util.Log

//CapMap holds a maping of methods to capabilities
var CapMap = make(map[string]*Capability)

// RC is the root capability
var RC = Capability{Name: "RootCapability"}

// Capability represents a security capability in the system
type Capability struct {
	Name   string
	Parent *Capability
	Tokens []Token
}

// Token is a unique id that represents a capability
type Token struct {
	ID string
}

// New returns a new capability
func New(name string) *Capability {
	return &Capability{Name: name, Tokens: []Token{}}
}

// SetParent takes a capability and sets it as the parent
func (cap *Capability) SetParent(parent *Capability) {
	cap.Parent = parent
}

// CreateToken creates a token for the respective capability
func (cap *Capability) CreateToken() Token {
	token := Token{ID: xid.New().String()}
	cap.Tokens = append(cap.Tokens, token)
	return token
}

// ValidateToken checks if the provided token belongs to the respective capability
func (cap *Capability) ValidateToken(apptoken Token) bool {
	for _, token := range cap.Tokens {
		if apptoken.ID == token.ID {
			log.Debug("Matched capability at " + cap.Name)
			return true
		}
	}
	if cap.Parent != nil {
		return cap.Parent.ValidateToken(apptoken)
	}
	return false
}

// SetMethodCap adds a capability for a specific method
func SetMethodCap(method string, cap *Capability) {
	log.Debugf("Setting capability %s for method %s", cap.Name, method)
	CapMap[method] = cap
}

// GetMethodCap returns a capability for a specific method
func GetMethodCap(method string) (*Capability, error) {
	if cap, ok := CapMap[method]; ok {
		return cap, nil
	}
	return nil, errors.New("Can't find capability for method " + method)
}

// GetMethodName returns a string representation of the passed method
func GetMethodName(method interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(method).Pointer()).Name()
}
