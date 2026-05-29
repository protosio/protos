#pragma once

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

char *ProtosStart(char *configJSON);
ProtosResult ProtosCall(char *method, void *request, int64_t requestLen);
ProtosWatchResult ProtosWatchChanges(void *request, int64_t requestLen, void *callbackContext, ProtosWatchCallback callback);
ProtosWatchResult ProtosWatchChangesBytes(void *request, int64_t requestLen, void *callbackContext, ProtosWatchBytesCallback callback);
void ProtosCancelWatch(int64_t watchID);
char *ProtosStop(void);
void ProtosFree(void *ptr);
