package capability

import (
	"errors"
	"protos/database"
	"protos/util"
	"reflect"
	"runtime"

	"github.com/asdine/storm"
	"github.com/rs/xid"
)

var log = util.Log

//CapMap holds a maping of methods to capabilities
var CapMap = make(map[string]*Capability)

// RC is the root capability
var RC *Capability

// Capability represents a security capability in the system
type Capability struct {
	Name   string `storm:"id"`
	Parent *Capability
	Tokens []Token
}

// Token is a unique id that represents a capability
type Token struct {
	ID string
}

// Initialize creates the root capability and retrieves any tokens that are stored in the db
func Initialize() {
	RC = New("RootCapability")
}

// New returns a new capability
func New(name string) *Capability {
	log.Debugf("Creating capability %s", name)
	capa := Capability{}
	err := database.One("Name", name, &capa)
	if err == storm.ErrNotFound {
		capa.Name = name
		capa.Tokens = []Token{}
	} else if err != nil {
		log.Fatal(err)
	}
	return &capa
}

// Save persists a capability and all it's tokens to databse
func (cap *Capability) Save() {
	log.Debugf("Saving capability %s", cap.Name)
	err := database.Save(cap)
	if err != nil {
		log.Fatal(err)
	}
}

// SetParent takes a capability and sets it as the parent
func (cap *Capability) SetParent(parent *Capability) {
	cap.Parent = parent
}

// CreateToken creates a token for the respective capability
func (cap *Capability) CreateToken() Token {
	token := Token{ID: xid.New().String()}
	cap.Tokens = append(cap.Tokens, token)
	cap.Save()
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
