package daemon

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cnf/structhash"
	"github.com/tidwall/gjson"
)

type DNSResource struct {
	Host  string `json:"host"`
	Value string `json:"value" hash:"-"`
	Type  string `json:"type"`
	TTL   int    `json:"ttl" hash:"-"`
}

type Resource struct {
	ID     string      `json:"id" hash:"-"`
	Type   string      `json:"type"`
	Fields interface{} `json:"value"`
	Status string      `json:"status"`
	App    *App        `json:"app" hash:"-"`
}

var resources = make(map[string]Resource)

// Provider defines a Protos resource provider
type Provider struct {
	Type string
	App  *App
}

var providers = make(map[string]Provider)

//
// Providers
//

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
	err := errors.New("Application '" + app.Name + "' is NOT registered as a resource provider.")
	log.Error(err)
	return map[string]Resource{}, err
}

//
// Resource
//

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

//GetResources retrieves all the saved resources
// some fields are modified before being returned
func GetResources() map[string]interface{} {
	type Alias Resource
	type MResource struct {
		App string `json:"app"`
		*Alias
	}
	modifiedResources := make(map[string]interface{})
	for id, rsc := range resources {
		mrsc := MResource{
			App:   rsc.App.ID,
			Alias: (*Alias)(&rsc),
		}
		modifiedResources[id] = mrsc
	}
	return modifiedResources
}

// GetAppResources retrieves all the resources that belong to an application
func GetAppResources(app *App) map[string]Resource {
	rsc := make(map[string]Resource)
	for id, resource := range resources {
		if resource.App.ID == app.ID {
			rsc[id] = resource
		}
	}
	return rsc
}

//CreateResource adds a resource to the internal resources map.
func CreateResource(appJSON []byte, appIP string) (Resource, error) {
	app, err := ReadAppByIP(appIP)
	if err != nil {
		return Resource{}, err
	}

	resource, err := GetResourceFromJSON(appJSON)
	if err != nil {
		return Resource{}, err
	}

	if IsValidResourceType(resource.Type) == false {
		return Resource{}, errors.New("Resource type '" + resource.Type + "' is invalid.")
	}

	rhash := fmt.Sprintf("%x", structhash.Md5(resource, 1))
	if rsc, ok := resources[rhash]; ok {
		return Resource{}, errors.New("Resource " + rhash + " already registered for application " + rsc.App.Name)
	}
	log.Debug("Adding resource ", rhash, ": ", resource)

	resource.App = &app
	resource.Status = "registered"
	resource.ID = rhash
	resources[rhash] = resource
	return resource, nil
}

//DeleteResource deletes a resource
func DeleteResource(resourceID string, appIP string) error {
	resource, ok := resources[resourceID]
	if ok != true {
		return errors.New("Resource " + resourceID + " does not exist.")
	}

	app, err := ReadAppByIP(appIP)
	if err != nil {
		return err
	}

	if resource.App.ID != app.ID {
		return errors.New("Resource " + resourceID + " not owned by application " + app.ID)
	}
	log.Info("Deleting resource " + resourceID + " belonging to application " + resource.App.ID)
	delete(resources, resourceID)
	return nil
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
