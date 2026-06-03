//go:build darwin || ios

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
typedef void (*ProtosWatchBytesCallback)(void *context, void *data, int64_t len, char *err);

static inline void ProtosInvokeWatchCallback(ProtosWatchCallback callback, void *context, ProtosResult result) {
	callback(context, result);
}

static inline void ProtosInvokeWatchBytesCallback(ProtosWatchBytesCallback callback, void *context, void *data, int64_t len, char *err) {
	callback(context, data, len, err);
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
	watches  = map[int64]*watchEntry{}
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
		cancelAllWatches()
		return nil
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
	return startWatch(request, requestLen, func(response []byte) {
		C.ProtosInvokeWatchCallback(callback, callbackContext, bytesResult(response))
	}, func(err error) {
		C.ProtosInvokeWatchCallback(callback, callbackContext, errorResult(err))
	}, func() {
		C.ProtosInvokeWatchCallback(callback, callbackContext, C.ProtosResult{})
	})
}

//export ProtosWatchChangesBytes
func ProtosWatchChangesBytes(request unsafe.Pointer, requestLen C.int64_t, callbackContext unsafe.Pointer, callback C.ProtosWatchBytesCallback) C.ProtosWatchResult {
	if callback == nil {
		return watchErrorResult(fmt.Errorf("watch callback is required"))
	}
	return startWatch(request, requestLen, func(response []byte) {
		result := bytesResult(response)
		C.ProtosInvokeWatchBytesCallback(callback, callbackContext, result.data, result.len, result.err)
	}, func(err error) {
		result := errorResult(err)
		C.ProtosInvokeWatchBytesCallback(callback, callbackContext, result.data, result.len, result.err)
	}, func() {
		C.ProtosInvokeWatchBytesCallback(callback, callbackContext, nil, 0, nil)
	})
}

func startWatch(
	request unsafe.Pointer,
	requestLen C.int64_t,
	emitData func([]byte),
	emitError func(error),
	emitDone func(),
) C.ProtosWatchResult {
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
	entry := &watchEntry{cancel: cancel, active: true}
	watchMu.Lock()
	watches[id] = entry
	watchMu.Unlock()

	go func() {
		defer unregisterWatch(id)
		err := current.WatchChanges(ctx, payload, func(response []byte) bool {
			return entry.emit(ctx, func() {
				emitData(response)
			})
		})
		if err != nil {
			entry.emitTerminal(ctx, func() {
				emitError(err)
			})
			return
		}
		entry.emitTerminal(ctx, emitDone)
	}()

	return C.ProtosWatchResult{watch_id: C.int64_t(id)}
}

//export ProtosCancelWatch
func ProtosCancelWatch(id C.int64_t) {
	cancelWatch(int64(id))
}

//export ProtosCancelAllWatches
func ProtosCancelAllWatches() {
	cancelAllWatches()
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
	entry := watches[id]
	delete(watches, id)
	watchMu.Unlock()
	if entry != nil {
		entry.cancelWatch()
	}
}

func cancelAllWatches() {
	watchMu.Lock()
	entries := make([]*watchEntry, 0, len(watches))
	for id, entry := range watches {
		entries = append(entries, entry)
		delete(watches, id)
	}
	watchMu.Unlock()
	for _, entry := range entries {
		entry.cancelWatch()
	}
}

func unregisterWatch(id int64) {
	watchMu.Lock()
	delete(watches, id)
	watchMu.Unlock()
}

type watchEntry struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	active bool
}

func (e *watchEntry) emit(ctx context.Context, emit func()) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.active || ctx.Err() != nil {
		return false
	}
	emit()
	return e.active && ctx.Err() == nil
}

func (e *watchEntry) emitTerminal(ctx context.Context, emit func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.active || ctx.Err() != nil {
		return
	}
	e.active = false
	emit()
}

func (e *watchEntry) cancelWatch() {
	e.mu.Lock()
	e.active = false
	cancel := e.cancel
	e.cancel = nil
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
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
