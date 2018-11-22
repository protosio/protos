package app

import (
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/util"
)

// mapps maintains a map of all the applications, which are all mutable
var mapps map[string]*App

// papps maintains a map of all applications, which are immutable and and atomically swapped
var papps map[string]App

type readAppResp struct {
	app interface{}
	err error
}

type readAppReq struct {
	id   string
	copy bool
	resp chan readAppResp
}

type removeAppReq struct {
	id   string
	resp chan error
}

// readAppQueue receives read requests for specific apps, based on app id
var readAppQueue = make(chan readAppReq)

// addAppQueue receives apps and adds them to the app list
var addAppQueue = make(chan *App, 100)

// saveAppSignal receives notifications to save the app to the db
var saveAppSignal = make(chan string, 100)

// removeAppQueue receives remove requests for specific apps, based on app id
var removeAppQueue = make(chan removeAppReq, 100)

// readAllQueue receives read requests for the whole app list
var readAllQueue = make(chan chan map[string]*App)

// readAllPublicQueue receives read requests for the whole immutable app list
var readAllPublicQueue = make(chan chan map[string]App)

// initDB runs when Protos starts and loads all apps from the DB in memory
func initDB() {
	log.WithField("proc", "appmanager").Debug("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&platform.DockerContainer{})

	dbapps := []*App{}
	err := database.All(&dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	mapps = make(map[string]*App)
	papps = make(map[string]App)
	for _, app := range dbapps {
		papps[app.ID] = *app
		tmp := app
		tmp.access = &sync.Mutex{}
		mapps[app.ID] = tmp
	}
}

func saveApp(app *App) {
	app.access.Lock()
	papp := *app
	app.access.Unlock()
	papp.access = nil
	papps[papp.ID] = papp
	gconfig.WSPublish <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeApp, PayloadValue: papp.Public()}
	err := database.Save(&papp)
	if err != nil {
		log.Panic(errors.Wrap(err, "Could not save app to database"))
	}
}

// Manager runs in its own goroutine and manages access to the app list
func Manager(quit chan bool) {
	log.WithField("proc", "appmanager").Info("Starting the app manager")
	initDB()
	for {
		select {
		case readReq := <-readAppQueue:
			var app interface{}
			var found bool
			if readReq.copy {
				app, found = papps[readReq.id]
			} else {
				app, found = mapps[readReq.id]
			}
			if found {
				readReq.resp <- readAppResp{app: app}
			} else {
				readReq.resp <- readAppResp{err: fmt.Errorf("Could not find app %s", readReq.id)}
			}
		case app := <-addAppQueue:
			mapps[app.ID] = app
			saveApp(app)
		case appID := <-saveAppSignal:
			saveApp(mapps[appID])
		case removeReq := <-removeAppQueue:
			if app, found := mapps[removeReq.id]; found {
				delete(mapps, app.ID)
				delete(papps, app.ID)
				err := database.Remove(app)
				if err != nil {
					removeReq.resp <- err
				}
				removeReq.resp <- nil
			} else {
				removeReq.resp <- fmt.Errorf("Could not find app %s", removeReq.id)
			}
		case readAllResp := <-readAllQueue:
			appsCopy := make(map[string]*App)
			for k, v := range mapps {
				appsCopy[k] = v
			}
			readAllResp <- appsCopy
		case readAllPublicResp := <-readAllPublicQueue:
			readAllPublicResp <- papps
		case <-quit:
			log.Info("Shutting down app manager")
			return
		}
	}
}

func addToManager(app *App) {
	addAppQueue <- app
}

// GetCopy returns a copy of application based on its id
func GetCopy(id string) (App, error) {
	log.Info("Reading application ", id)
	ra := readAppReq{id: id, copy: true, resp: make(chan readAppResp)}
	readAppQueue <- ra
	resp := <-ra.resp
	app := resp.app.(App)
	return app, resp.err
}
