package app

import (
	"encoding/gob"
	"fmt"

	"github.com/pkg/errors"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/util"
)

// mapps maintains a map of all the applications
var mapps map[string]App

type readAppResp struct {
	app App
	err error
}

type readAppReq struct {
	id   string
	resp chan readAppResp
}

type removeAppReq struct {
	id   string
	resp chan error
}

// readAppQueue receives read requests for specific apps, based on app id
var readAppQueue = make(chan readAppReq)

// addAppQueue receives apps and adds them to the app list
var addAppQueue = make(chan App, 100)

// removeAppQueue receives remove requests for specific apps, based on app id
var removeAppQueue = make(chan removeAppReq, 100)

// readAllQueue receives read requests for the whole app list
var readAllQueue = make(chan chan map[string]App)

// initDB runs when Protos starts and loads all apps from the DB in memory
func initDB() map[string]App {
	log.WithField("proc", "appmanager").Debug("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&platform.DockerContainer{})

	dbapps := []App{}
	err := database.All(&dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	lapps := make(map[string]App)
	for _, app := range dbapps {
		lapps[app.ID] = app
	}
	return lapps
}

// Manager runs in its own goroutine and manages access to the app list
func Manager(quit chan bool) {
	log.WithField("proc", "appmanager").Info("Starting the app manager")
	mapps = initDB()
	for {
		select {
		case readReq := <-readAppQueue:
			if app, found := mapps[readReq.id]; found {
				readReq.resp <- readAppResp{app: app}
			} else {
				readReq.resp <- readAppResp{err: fmt.Errorf("Could not find app %s", readReq.id)}
			}
		case app := <-addAppQueue:
			mapps[app.ID] = app
			gconfig.WSPublish <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeApp, PayloadValue: app}
			err := database.Save(&app)
			if err != nil {
				log.Panic(errors.Wrap(err, "Could not save app to database"))
			}
		case removeReq := <-removeAppQueue:
			if app, found := mapps[removeReq.id]; found {
				delete(mapps, app.ID)
				err := database.Remove(&app)
				if err != nil {
					removeReq.resp <- err
				}
				removeReq.resp <- nil
			} else {
				removeReq.resp <- fmt.Errorf("Could not find app %s", removeReq.id)
			}
		case readAllResp := <-readAllQueue:
			appsCopy := make(map[string]App)
			for k, v := range mapps {
				appsCopy[k] = v
			}
			readAllResp <- appsCopy
		case <-quit:
			log.Info("Shutting down app manager")
			return
		}
	}
}
