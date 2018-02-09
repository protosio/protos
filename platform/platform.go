package platform

import "github.com/nustiueudinastea/protos/util"

var log = util.Log

// RuntimeUnit represents the abstract concept of a running program: it can be a container, VM or process.
type RuntimeUnit interface {
	Start() error
	Stop() error
	Update() error
	Remove() error
}
