package app

import (
	"github.com/protosio/protos/internal/core"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

// taskMap is a local type that implements MarshalJSON interface
type taskMap linkedhashmap.Map

// Public returns a public version of the app struct
func (app App) Public() core.App {
	app.enrichAppData()
	return &app
}

// GetAllPublic returns all applications in their public form, enriched with the latest status message
func (am *Manager) GetAllPublic() map[string]core.App {
	// ToDo: do app refresh caching in the platform code
	papps := map[string]core.App{}

	for _, app := range am.apps.copy() {
		tmp := app
		tmp.enrichAppData()
		papps[tmp.ID] = &tmp
	}
	return papps
}

func (tm taskMap) MarshalJSON() ([]byte, error) {
	lhm := linkedhashmap.Map(tm)
	return lhm.ToJSON()
}
