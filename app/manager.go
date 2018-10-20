package app

import (
	"encoding/gob"
	"fmt"

	"github.com/protosio/protos/database"
	"github.com/protosio/protos/platform"
)

// apps maintains a map of all the applications
var apps = make(map[string]App)

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
func initDB() {
	log.WithField("proc", "appmanager").Debug("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&platform.DockerContainer{})

	dbapps := []App{}
	err := database.All(&dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from the database: ", err)
	}

	for _, app := range dbapps {
		apps[app.ID] = app
	}
}

// Manager runs in its own goroutine and manages access to the app list
func Manager(quit chan bool) {
	log.WithField("proc", "appmanager").Info("Starting the app manager")
	initDB()
	for {
		select {
		case readReq := <-readAppQueue:
			if app, found := apps[readReq.id]; found {
				readReq.resp <- readAppResp{app: app}
			} else {
				readReq.resp <- readAppResp{err: fmt.Errorf("Could not find app %s", readReq.id)}
			}
		case app := <-addAppQueue:
			apps[app.ID] = app
		case removeReq := <-removeAppQueue:
			if app, found := apps[removeReq.id]; found {
				delete(apps, app.ID)
				removeReq.resp <- nil
			} else {
				removeReq.resp <- fmt.Errorf("Could not find app %s", removeReq.id)
			}
		case readAllResp := <-readAllQueue:
			readAllResp <- apps
		case <-quit:
			log.Info("Shutting down app manager")
			return
		}
	}
}
