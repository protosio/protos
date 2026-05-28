//go:build darwin

package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void *data;
	int64_t len;
	char *err;
} ProtosResult;

typedef struct {
	int64_t watch_id;
	char *err;
} ProtosWatchResult;

typedef void (*ProtosWatchCallback)(void *context, ProtosResult result);

static inline void ProtosInvokeWatchCallback(ProtosWatchCallback callback, void *context, ProtosResult result) {
	callback(context, result);
}
*/
import "C"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/protosio/protos/internal/apibridge"
)

var (
	bridgeMu sync.Mutex
	bridge   *apibridge.Bridge
	watchMu  sync.Mutex
	watches  = map[int64]context.CancelFunc{}
	watchID  atomic.Int64
)

func main() {}

//export ProtosStart
func ProtosStart(configJSON *C.char) *C.char {
	var rawConfig []byte
	if configJSON != nil {
		rawConfig = []byte(C.GoString(configJSON))
	}

	bridgeMu.Lock()
	defer bridgeMu.Unlock()
	if bridge != nil {
		return C.CString("Protos daemon is already started")
	}
	started, err := apibridge.Start(context.Background(), rawConfig)
	if err != nil {
		return C.CString(err.Error())
	}
	bridge = started
	return nil
}

//export ProtosCall
func ProtosCall(method *C.char, request unsafe.Pointer, requestLen C.int64_t) C.ProtosResult {
	if method == nil {
		return errorResult(fmt.Errorf("API method is required"))
	}
	if requestLen < 0 {
		return errorResult(fmt.Errorf("request length cannot be negative"))
	}

	var payload []byte
	if request != nil && requestLen > 0 {
		payload = append([]byte(nil), unsafe.Slice((*byte)(request), int(requestLen))...)
	}

	bridgeMu.Lock()
	current := bridge
	bridgeMu.Unlock()
	if current == nil {
		return errorResult(fmt.Errorf("Protos daemon is not started"))
	}

	response, err := current.Call(context.Background(), C.GoString(method), payload)
	if err != nil {
		return errorResult(err)
	}
	return bytesResult(response)
}

//export ProtosWatchChanges
func ProtosWatchChanges(request unsafe.Pointer, requestLen C.int64_t, callbackContext unsafe.Pointer, callback C.ProtosWatchCallback) C.ProtosWatchResult {
	if callback == nil {
		return watchErrorResult(fmt.Errorf("watch callback is required"))
	}
	if requestLen < 0 {
		return watchErrorResult(fmt.Errorf("request length cannot be negative"))
	}

	var payload []byte
	if request != nil && requestLen > 0 {
		payload = append([]byte(nil), unsafe.Slice((*byte)(request), int(requestLen))...)
	}

	bridgeMu.Lock()
	current := bridge
	bridgeMu.Unlock()
	if current == nil {
		return watchErrorResult(fmt.Errorf("Protos daemon is not started"))
	}

	id := watchID.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	watchMu.Lock()
	watches[id] = cancel
	watchMu.Unlock()

	go func() {
		defer unregisterWatch(id)
		err := current.WatchChanges(ctx, payload, func(response []byte) bool {
			C.ProtosInvokeWatchCallback(callback, callbackContext, bytesResult(response))
			return ctx.Err() == nil
		})
		if err != nil {
			C.ProtosInvokeWatchCallback(callback, callbackContext, errorResult(err))
			return
		}
		C.ProtosInvokeWatchCallback(callback, callbackContext, C.ProtosResult{})
	}()

	return C.ProtosWatchResult{watch_id: C.int64_t(id)}
}

//export ProtosCancelWatch
func ProtosCancelWatch(id C.int64_t) {
	cancelWatch(int64(id))
}

//export ProtosStop
func ProtosStop() *C.char {
	cancelAllWatches()
	bridgeMu.Lock()
	current := bridge
	bridge = nil
	bridgeMu.Unlock()
	if current == nil {
		return nil
	}
	if err := current.Stop(); err != nil {
		return C.CString(err.Error())
	}
	return nil
}

func cancelWatch(id int64) {
	watchMu.Lock()
	cancel := watches[id]
	delete(watches, id)
	watchMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func cancelAllWatches() {
	watchMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(watches))
	for id, cancel := range watches {
		cancels = append(cancels, cancel)
		delete(watches, id)
	}
	watchMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func unregisterWatch(id int64) {
	watchMu.Lock()
	delete(watches, id)
	watchMu.Unlock()
}

//export ProtosFree
func ProtosFree(ptr unsafe.Pointer) {
	if ptr != nil {
		C.free(ptr)
	}
}

func bytesResult(data []byte) C.ProtosResult {
	if len(data) == 0 {
		return C.ProtosResult{}
	}
	ptr := C.CBytes(data)
	return C.ProtosResult{
		data: ptr,
		len:  C.int64_t(len(data)),
	}
}

func errorResult(err error) C.ProtosResult {
	if err == nil {
		return C.ProtosResult{}
	}
	return C.ProtosResult{err: C.CString(err.Error())}
}

func watchErrorResult(err error) C.ProtosWatchResult {
	if err == nil {
		return C.ProtosWatchResult{}
	}
	return C.ProtosWatchResult{err: C.CString(err.Error())}
}
