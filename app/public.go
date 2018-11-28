package app

import (
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/protosio/protos/task"
	"github.com/protosio/protos/util"
)

type taskMap linkedhashmap.Map

// PublicApp is used for marshalling app data to the UI
type PublicApp struct {
	App
	Tasks taskMap `json:"tasks"`
}

// ToDo: do app refresh caching in the platform code
func enrichPublicApps(apps map[string]App) map[string]PublicApp {
	tasks := task.GetAll()
	papps := map[string]PublicApp{}

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
		papps[papp.ID] = papp
	}
	return papps
}

// Public returns a public version of the app struct
func (app App) Public() PublicApp {
	app.enrichAppData()
	pa := PublicApp{
		App: app,
	}
	pa.Tasks = taskMap(task.GetIDs(app.Tasks))
	return pa
}

// GetAllPublic returns all applications in their public form, enriched with the latest status message
func GetAllPublic() map[string]PublicApp {
	papps := enrichPublicApps(CopyAll())
	return papps
}

func (tm taskMap) MarshalJSON() ([]byte, error) {
	lhm := linkedhashmap.Map(tm)
	return lhm.ToJSON()
}
