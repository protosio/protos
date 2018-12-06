package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var externalWSRoutes = routes{
	route{
		Name:        "WebSockets",
		Pattern:     "/ws",
		HandlerFunc: ws,
	},
}

type wsConnection struct {
	Send chan interface{}
}

var upgrader = websocket.Upgrader{}
var wsConnections = map[string]wsConnection{}
var addWSQueue = make(chan wsConnection, 100)
var removeWSQueue = make(chan wsConnection, 100)

//
// WS connection manager
//

// WSManager is the WS connection manager that owns the WS connection data
func WSManager(quit chan bool) {
	log.WithField("proc", "wsmanager").Info("Starting the WS connection manager")

	for {
		select {
		case wsCon := <-addWSQueue:
			conID := fmt.Sprintf("%p", &wsCon)
			log.Debug("Registering WS connection ", conID)
			wsConnections[conID] = wsCon
		case wsCon := <-removeWSQueue:
			conID := fmt.Sprintf("%p", &wsCon)
			log.Debug("Deregistering WS connection ", conID)
			delete(wsConnections, conID)
		case publishMsg := <-gconfig.WSPublish:
			for _, wsCon := range wsConnections {
				wsCon.Send <- publishMsg
			}
		case <-quit:
			log.Info("Shutting down WS connection manager")
			return
		}
	}

}

//
// API endpoint
//

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Error upgrading websocket connection: ", err)
		return
	}
	log.Debugf("Upgraded websocket connection %p", &c)
	wsCon := wsConnection{Send: make(chan interface{}, 100)}
	addWSQueue <- wsCon
	for {
		select {
		case msg := <-wsCon.Send:
			log.Debugf("Writing to websocket connection %p: %v", &wsCon, msg)
			jsonMsg, err := json.Marshal(msg)
			if err != nil {
				log.Errorf("Failed to marshall ws message struct %v: %s", msg, err)
				continue
			}
			err = c.WriteMessage(1, jsonMsg)
			if err != nil {
				log.Errorf("Error writing to websocket connection %p: %s", &wsCon, err)
				c.Close()
				removeWSQueue <- wsCon
				return
			}
		}
	}
}
