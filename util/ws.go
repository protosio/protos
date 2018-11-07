package util

const (
	// WSMsgTypeUpdate represents an message that contains an update for a specific Protos entity (app, task, resource)
	WSMsgTypeUpdate = "update"

	// WSPayloadTypeTask represents a task
	WSPayloadTypeTask = "task"
	// WSPayloadTypeApp represents an app
	WSPayloadTypeApp = "app"
	// WSPayloadTypeResource represents a resource
	WSPayloadTypeResource = "resource"
)

// WSMessage is used as a container struct sent as a message with a payload to thw WS clients
type WSMessage struct {
	MsgType      string
	PayloadType  string
	PayloadValue interface{}
}
