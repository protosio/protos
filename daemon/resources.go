package daemon

import (
	"encoding/json"
	"errors"

	"github.com/cnf/structhash"
	"github.com/tidwall/gjson"
)

type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

// Resource defines a Protos resource
// type Resource struct {
// 	Type   string            `json:"type"`
// 	Fields map[string]string `json:"value"`
// 	Status string            `json:"status"`
// 	App    *App              `json:"app"`
// }

type Resource struct {
	Type   string      `json:"type"`
	Fields interface{} `json:"value"`
	Status string      `json:"status"`
	App    *App        `json:"app"`
}

var resources = make(map[string]Resource)

// Provider defines a Protos resource provider
type Provider struct {
	Type string
	App  *App
}

var providers = make(map[string]Provider)

//IsValidResourceType check if a resource type is valid
func IsValidResourceType(rtype string) bool {
	switch rtype {
	case
		"dns",
		"mail":
		return true
	}
	return false
}

// RegisterProvider registers a resource provider
func RegisterProvider(provider Provider, app *App) error {
	if IsValidResourceType(provider.Type) == false {
		log.Error("Resource type '", provider.Type, "' is invalid.")
		return errors.New("Resource type '" + provider.Type + "' is invalid.")
	}
	if val, ok := providers[provider.Type]; ok {
		err := errors.New("Application '" + val.App.Name + "' already registered for resource type '" + provider.Type + "'.")
		log.Error(err)
		return err
	}

	log.Info("Registering application '" + app.Name + "' as a '" + provider.Type + "' provider.")
	provider.App = app
	providers[provider.Type] = provider
	return nil
}

// UnregisterProvider unregisters a resource provider
func UnregisterProvider(provider Provider, app *App) error {
	if IsValidResourceType(provider.Type) == false {
		log.Error("Resource type '", provider.Type, "' is invalid.")
		return errors.New("Resource type '" + provider.Type + "' is invalid.")
	}
	if _, ok := providers[provider.Type]; ok {
		log.Info("Unregistering application '" + app.Name + "' as a '" + provider.Type + "' provider.")
		delete(providers, provider.Type)
		return nil
	}

	err := errors.New("Application '" + app.Name + "' is NOT registered for resource type '" + provider.Type + "'.")
	log.Error(err)
	return err
}

//GetProviderResources retrieves all resources of a specific resource provider.
func GetProviderResources(app *App) (map[string]Resource, error) {
	for _, provider := range providers {
		if provider.App.ID == app.ID {
			res := map[string]Resource{}
			for id, resource := range resources {
				if provider.Type == resource.Type {
					res[id] = resource
				}
			}
			return res, nil
		}
	}
	err := errors.New("Application '" + app.Name + "' is NOT registered as a resource resource provider.")
	log.Error(err)
	return map[string]Resource{}, err
}

//AddResource adds a resource to the internal resources map.
func AddResource(resource Resource, app *App) error {
	if IsValidResourceType(resource.Type) == false {
		log.Error("Resource type '", resource.Type, "' is invalid.")
		return errors.New("Resource type '" + resource.Type + "' is invalid.")
	}
	rhash, err := structhash.Hash(resource, 1)
	if err != nil {
		log.Error(err)
		return err
	}

	resource.App = app
	resource.Status = "registered"
	log.Debug("Adding resource ", rhash, ": ", resource)
	resources[rhash] = resource
	return nil
}

//GetResources retrieves all the saved resources
func GetResources() []Resource {
	rsc := []Resource{}
	for _, resource := range resources {
		rsc = append(rsc, resource)
	}
	return rsc
}

//GetResourceFromJSON recevies json and casts it to the correct data structure
func GetResourceFromJSON(resourceJSON []byte) (Resource, error) {

	resource := Resource{}
	err := json.Unmarshal(resourceJSON, &resource)
	if err != nil {
		return Resource{}, err
	}

	resourceJSONValue := gjson.GetBytes(resourceJSON, "value")
	var raw []byte
	if resourceJSONValue.Index > 0 {
		raw = resourceJSON[resourceJSONValue.Index : resourceJSONValue.Index+len(resourceJSONValue.Raw)]
	} else {
		raw = []byte(resourceJSONValue.Raw)
	}

	resourceType := gjson.Get(string(resourceJSON), "type").Str
	if resourceType == "dns" {
		resourceStruct := DNSResource{}
		err = json.Unmarshal(raw, &resourceStruct)
		if err != nil {
			return Resource{}, err
		}
		resource.Fields = resourceStruct
	} else {
		return Resource{}, errors.New("Resource type '" + resourceType + "' does not exists")
	}

	return resource, nil
}
