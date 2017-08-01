package daemon

import (
	"github.com/cnf/structhash"
)

// Resource defines a Protos resource
type Resource struct {
	Type   string            `json:"type"`
	Fields map[string]string `json:"value"`
}

var resources = make(map[string]Resource)

//AddResources adds a resource to the internal resources map.
func AddResources(resource Resource) error {
	rhash, err := structhash.Hash(resource, 1)
	if err != nil {
		log.Error(err)
		return err
	}
	log.Debug("Adding resource ", rhash, ": ", resource)
	resources[rhash] = resource
	return nil
}
