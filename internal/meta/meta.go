package meta

import (
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("meta")
var gconfig = config.Get()

// Meta contains information about the Protos instance
type Meta struct {
	db      *db.DB
	version string

	// Public members
	ID           string
	InstanceName string
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup(db *db.DB, keymngr *pcrypto.Manager, version string) *Meta {
	if db == nil || keymngr == nil {
		log.Panic("Failed to setup meta package: none of the inputs can be nil")
	}

	metaRoot := Meta{}
	log.Debug("Reading instance information from database")

	metaRoot.db = db
	metaRoot.version = version
	return &metaRoot
}

// SetInstanceName sets the name of the instance
func (m *Meta) SetInstanceName(name string) {
	log.Debugf("Setting instance name to '%s'", name)
	m.InstanceName = name
}

// GetInstanceName retrieves the name of the instance
func (m *Meta) GetInstanceName() string {
	return m.InstanceName
}

// GetVersion returns current version
func (m *Meta) GetVersion() string {
	return m.version
}
