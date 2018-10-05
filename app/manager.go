package app

import "fmt"

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

// Manager runs in its own goroutine and manages access to the app list
func Manager() {
	log.WithField("proc", "appmanager").Info("Starting the app manager")
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
		}
	}
}
