package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/protosio/protos/internal/core"

	"github.com/gorilla/websocket"
)

var externalWSRoutes = routes{
	route{
		Name:        "ExternalWebSockets",
		Pattern:     "/ws",
		HandlerFunc: wsExternal,
	},
}

var internalWSRoutes = routes{
	route{
		Name:        "InternalWebSockets",
		Pattern:     "/ws",
		HandlerFunc: wsInternal,
	},
}

type wsConnection struct {
	Send  chan interface{}
	Close chan bool
}

var upgrader = websocket.Upgrader{}
var wsConnections = map[string]*wsConnection{}
var addWSQueue = make(chan *wsConnection, 100)
var removeWSQueue = make(chan *wsConnection, 100)

//
// WS connection manager
//

// WSManager is the WS connection manager that owns the WS connection data
func WSManager(am core.AppManager, quit chan bool, wsfrontend chan interface{}) {
	log.WithField("proc", "wsmanager").Info("Starting the WS connection manager")

	for {
		select {
		case wsCon := <-addWSQueue:
			conID := fmt.Sprintf("%p", wsCon)
			log.WithField("proc", "wsmanager").Debugf("Registering WS connection '%s'", conID)
			wsConnections[conID] = wsCon
		case wsCon := <-removeWSQueue:
			conID := fmt.Sprintf("%p", wsCon)
			log.WithField("proc", "wsmanager").Debugf("Deregistering WS connection '%s'", conID)
			delete(wsConnections, conID)
		case publishMsg := <-wsfrontend:
			for _, wsCon := range wsConnections {
				wsCon.Send <- publishMsg
			}
		case <-quit:
			log.WithField("proc", "wsmanager").Debug("Terminating all WS connections")

			// terminating frontend WS connections
			for _, wsCon := range wsConnections {
				wsCon.Close <- true
				conID := fmt.Sprintf("%p", wsCon)
				log.WithField("proc", "wsmanager").Debug("Deregistering external WS connection ", conID)
				delete(wsConnections, conID)
			}

			// terminating internal WS connections
			apps := am.CopyAll()
			for _, app := range apps {
				app.CloseMsgQ()
			}

			log.WithField("proc", "wsmanager").Info("Shutting down WS connection manager")
			return
		}
	}

}

func wsMessageReader(c *websocket.Conn, id string, quit chan bool) {
	// read message from client and process it
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "terminating") == false && strings.Contains(err.Error(), "use of closed network connection") == false {
				log.Errorf("Failed to read from WS connection %s: %s", id, err.Error())
			}
			quit <- true
			return
		}
	}
}

//
// External API endpoint
//

func wsExternal(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Error upgrading external websocket connection: ", err)
			return
		}
		wsCon := &wsConnection{Send: make(chan interface{}, 100), Close: make(chan bool, 1)}
		conID := fmt.Sprintf("%p", wsCon)
		remoteQuit := make(chan bool, 1)
		go wsMessageReader(c, conID, remoteQuit)
		addWSQueue <- wsCon
		log.Debugf("Upgraded external websocket connection '%s'", conID)
		for {
			select {
			case msg := <-wsCon.Send:
				log.Tracef("Writing to external websocket connection %s: %v", conID, msg)
				jsonMsg, err := json.Marshal(msg)
				if err != nil {
					log.Errorf("Failed to marshall ws message struct %v: %s", msg, err)
					continue
				}
				err = c.WriteMessage(1, jsonMsg)
				if err != nil {
					log.Errorf("Error writing to external websocket connection %s: %s", conID, err)
					c.Close()
					removeWSQueue <- wsCon
					return
				}
			case <-wsCon.Close:
				log.Debug("Closing external WS connection ", conID)
				c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "terminating"))
				c.Close()
				return
			case <-remoteQuit:
				removeWSQueue <- wsCon
				log.Debugf("WS connection %s remotely closed ", conID)
				return
			}
		}
	})
}

//
// Internal API endpoint
//

func wsInternal(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Error("Error upgrading websocket connection: ", err)
			return
		}

		ctx := r.Context()
		appInstance := ctx.Value(appKey).(core.App)
		msgq := &core.WSConnection{Send: make(chan interface{}, 100), Close: make(chan bool, 1)}
		appInstance.SetMsgQ(msgq)

		remoteQuit := make(chan bool, 1)
		conID := fmt.Sprintf("%p", msgq)
		go wsMessageReader(c, conID, remoteQuit)

		log.Debugf("Upgraded internal websocket connection %s", conID)
		for {
			select {
			case msg := <-msgq.Send:
				log.Debugf("Writing to internal websocket connection %s: %v", conID, msg)
				jsonMsg, err := json.Marshal(msg)
				if err != nil {
					log.Errorf("Failed to marshall ws message struct %v: %s", msg, err)
					continue
				}
				err = c.WriteMessage(1, jsonMsg)
				if err != nil {
					log.Errorf("Error writing to internal websocket connection %s: %s", conID, err)
					c.Close()
					appInstance.SetMsgQ(nil)
					return
				}
			// happens when Protos is shutting down and all WS connections need to terminate
			case <-msgq.Close:
				log.Debug("Closing WS connection ", conID)
				c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "terminating"))
				c.Close()
				return
			// happens when the connection has been closed from the other side and this routine needs to terminate
			case <-remoteQuit:
				appInstance.CloseMsgQ()
				log.Debugf("WS connection %s remotely closed ", conID)
				return
			}
		}
	})
}
