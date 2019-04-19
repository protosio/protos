package app

import (
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/protosio/protos/core"
	"github.com/protosio/protos/util"
)

type taskMap linkedhashmap.Map

// PublicApp is used for marshalling app data to the UI
type PublicApp struct {
	App

	Tasks     taskMap                  `json:"tasks"`
	Resources map[string]core.Resource `json:"resources"`
}

// ToDo: do app refresh caching in the platform code
func (am *Manager) enrichPublicApps(apps map[string]App) map[string]core.App {
	tasks := am.tm.GetAll()
	papps := map[string]core.App{}

	for _, app := range apps {
		tmp := app
		tmp.enrichAppData()
		papp := PublicApp{App: tmp}
		// using a closure to access the task ids stored in tmp.Tasks
		filter := func(k interface{}, v interface{}) bool {
			if found, _ := util.StringInSlice(k.(string), tmp.Tasks); found {
				return true
			}
			return false
		}
		papp.Tasks = taskMap(*tasks.Select(filter))
		papps[papp.ID] = &papp
	}
	return papps
}

// Public returns a public version of the app struct
func (app App) Public() PublicApp {
	app.enrichAppData()
	pa := PublicApp{
		App: app,
	}
	pa.Tasks = taskMap(app.parent.tm.GetIDs(app.Tasks))
	resourceFilter := func(rsc core.Resource) bool {
		found, _ := util.StringInSlice(rsc.GetID(), app.Resources)
		if found {
			return true
		}
		return false
	}
	pa.Resources = app.parent.rm.Select(resourceFilter)
	return pa
}

// GetAllPublic returns all applications in their public form, enriched with the latest status message
func (am *Manager) GetAllPublic() map[string]core.App {
	papps := am.enrichPublicApps(am.CopyAll())
	return papps
}

func (tm taskMap) MarshalJSON() ([]byte, error) {
	lhm := linkedhashmap.Map(tm)
	return lhm.ToJSON()
}
