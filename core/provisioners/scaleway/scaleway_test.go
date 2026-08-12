package scaleway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	block "github.com/scaleway/scaleway-sdk-go/api/block/v1"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	testScalewayZone     = "fr-par-1"
	testScalewayVolumeID = "volume-id"
	testScalewayServerID = "server-id"
)

func TestDeleteVolumeAttachedBlockDetachesBeforeRetryingBlockDelete(t *testing.T) {
	var mu sync.Mutex
	requests := make([]string, 0, 8)
	blockDeletes := 0
	blockGets := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodDelete && r.URL.Path == blockVolumePath():
			blockDeletes++
			if blockDeletes == 1 {
				writeScalewayError(w, http.StatusPreconditionFailed, "precondition_failed", "Can't delete this block volume, it is in_use status", "volume", testScalewayVolumeID)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == instanceVolumePath():
			t.Errorf("legacy Instance volume DELETE must not follow a non-not-found Block API error")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == instanceVolumePath():
			writeScalewayNotFound(w, "instance_volume", testScalewayVolumeID)
		case r.Method == http.MethodGet && r.URL.Path == blockVolumePath():
			blockGets++
			if blockGets == 1 {
				writeJSON(w, http.StatusOK, map[string]any{
					"id":     testScalewayVolumeID,
					"status": block.VolumeStatusInUse,
					"zone":   testScalewayZone,
					"references": []map[string]any{{
						"product_resource_type": "instance_server",
						"product_resource_id":   testScalewayServerID,
						"status":                block.ReferenceStatusAttached,
					}},
				})
				return
			}
			if blockGets == 2 {
				writeJSON(w, http.StatusOK, map[string]any{
					"id":         testScalewayVolumeID,
					"status":     block.VolumeStatusAvailable,
					"zone":       testScalewayZone,
					"references": []any{},
				})
				return
			}
			writeScalewayNotFound(w, "volume", testScalewayVolumeID)
		case r.Method == http.MethodPost && r.URL.Path == detachVolumePath():
			writeJSON(w, http.StatusOK, map[string]any{
				"server": map[string]any{"id": testScalewayServerID, "volumes": map[string]any{}},
			})
		default:
			t.Errorf("unexpected Scaleway request: %s %s", r.Method, r.URL.Path)
			writeScalewayNotFound(w, "test_route", r.URL.Path)
		}
	})

	sw := newTestScaleway(t, handler)
	if err := sw.DeleteVolume(testScalewayVolumeID, testScalewayZone); err != nil {
		t.Fatalf("DeleteVolume() error = %v", err)
	}

	want := []string{
		"DELETE " + blockVolumePath(),
		"GET " + blockVolumePath(),
		"POST " + detachVolumePath(),
		"GET " + instanceVolumePath(),
		"GET " + blockVolumePath(),
		"DELETE " + blockVolumePath(),
		"GET " + blockVolumePath(),
		"GET " + instanceVolumePath(),
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("Scaleway request sequence = %#v, want %#v", requests, want)
	}
}

func TestDeleteVolumePreservesLegacyInstanceVolumeFallback(t *testing.T) {
	var mu sync.Mutex
	requests := make([]string, 0, 10)
	instanceDeletes := 0
	instanceGets := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodDelete && r.URL.Path == blockVolumePath():
			writeScalewayNotFound(w, "volume", testScalewayVolumeID)
		case r.Method == http.MethodDelete && r.URL.Path == instanceVolumePath():
			instanceDeletes++
			if instanceDeletes == 1 {
				writeScalewayError(w, http.StatusPreconditionFailed, "precondition_failed", "resource is still in use, a server is attached to this volume", "instance_volume", testScalewayVolumeID)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == instanceVolumePath():
			instanceGets++
			if instanceGets == 1 {
				writeJSON(w, http.StatusOK, map[string]any{
					"volume": map[string]any{
						"id":          testScalewayVolumeID,
						"volume_type": instance.VolumeVolumeTypeLSSD,
						"state":       instance.VolumeStateAvailable,
						"server":      map[string]any{"id": testScalewayServerID},
						"zone":        testScalewayZone,
					},
				})
				return
			}
			if instanceGets == 2 {
				writeJSON(w, http.StatusOK, map[string]any{
					"volume": map[string]any{
						"id":          testScalewayVolumeID,
						"volume_type": instance.VolumeVolumeTypeLSSD,
						"state":       instance.VolumeStateAvailable,
						"zone":        testScalewayZone,
					},
				})
				return
			}
			writeScalewayNotFound(w, "instance_volume", testScalewayVolumeID)
		case r.Method == http.MethodGet && r.URL.Path == blockVolumePath():
			writeScalewayNotFound(w, "volume", testScalewayVolumeID)
		case r.Method == http.MethodPost && r.URL.Path == detachVolumePath():
			writeJSON(w, http.StatusOK, map[string]any{
				"server": map[string]any{"id": testScalewayServerID, "volumes": map[string]any{}},
			})
		default:
			t.Errorf("unexpected Scaleway request: %s %s", r.Method, r.URL.Path)
			writeScalewayNotFound(w, "test_route", r.URL.Path)
		}
	})

	sw := newTestScaleway(t, handler)
	if err := sw.DeleteVolume(testScalewayVolumeID, testScalewayZone); err != nil {
		t.Fatalf("DeleteVolume() error = %v", err)
	}

	want := []string{
		"DELETE " + blockVolumePath(),
		"DELETE " + instanceVolumePath(),
		"GET " + blockVolumePath(),
		"GET " + instanceVolumePath(),
		"POST " + detachVolumePath(),
		"GET " + instanceVolumePath(),
		"DELETE " + blockVolumePath(),
		"DELETE " + instanceVolumePath(),
		"GET " + blockVolumePath(),
		"GET " + instanceVolumePath(),
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("Scaleway request sequence = %#v, want %#v", requests, want)
	}
}

func TestVolumeDeletedRequiresAbsenceFromBothScalewayAPIs(t *testing.T) {
	tests := []struct {
		name           string
		blockExists    bool
		instanceExists bool
		wantDeleted    bool
	}{
		{name: "block volume remains", blockExists: true},
		{name: "legacy volume remains", instanceExists: true},
		{name: "both APIs report absent", wantDeleted: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == blockVolumePath():
					if test.blockExists {
						writeJSON(w, http.StatusOK, map[string]any{
							"id":     testScalewayVolumeID,
							"status": block.VolumeStatusAvailable,
							"zone":   testScalewayZone,
						})
						return
					}
					writeScalewayNotFound(w, "volume", testScalewayVolumeID)
				case r.Method == http.MethodGet && r.URL.Path == instanceVolumePath():
					if test.instanceExists {
						writeJSON(w, http.StatusOK, map[string]any{
							"volume": map[string]any{
								"id":          testScalewayVolumeID,
								"volume_type": instance.VolumeVolumeTypeLSSD,
								"state":       instance.VolumeStateAvailable,
								"zone":        testScalewayZone,
							},
						})
						return
					}
					writeScalewayNotFound(w, "instance_volume", testScalewayVolumeID)
				default:
					t.Errorf("unexpected Scaleway request: %s %s", r.Method, r.URL.Path)
					writeScalewayNotFound(w, "test_route", r.URL.Path)
				}
			})

			sw := newTestScaleway(t, handler)
			deleted, err := sw.volumeDeleted(testScalewayVolumeID, testScalewayZone)
			if err != nil {
				t.Fatalf("volumeDeleted() error = %v", err)
			}
			if deleted != test.wantDeleted {
				t.Fatalf("volumeDeleted() = %t, want %t", deleted, test.wantDeleted)
			}
		})
	}
}

func newTestScaleway(t *testing.T, handler http.Handler) *scaleway {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := scw.NewClient(
		scw.WithAPIURL(server.URL),
		scw.WithoutAuth(),
		scw.WithDefaultZone(scw.Zone(testScalewayZone)),
	)
	if err != nil {
		t.Fatalf("scw.NewClient() error = %v", err)
	}
	return &scaleway{
		client:      client,
		instanceAPI: instance.NewAPI(client),
		blockAPI:    block.NewAPI(client),
	}
}

func blockVolumePath() string {
	return fmt.Sprintf("/block/v1/zones/%s/volumes/%s", testScalewayZone, testScalewayVolumeID)
}

func instanceVolumePath() string {
	return fmt.Sprintf("/instance/v1/zones/%s/volumes/%s", testScalewayZone, testScalewayVolumeID)
}

func detachVolumePath() string {
	return fmt.Sprintf("/instance/v1/zones/%s/servers/%s/detach-volume", testScalewayZone, testScalewayServerID)
}

func writeScalewayNotFound(w http.ResponseWriter, resource string, resourceID string) {
	writeScalewayError(w, http.StatusNotFound, "not_found", "resource is not found", resource, resourceID)
}

func writeScalewayError(w http.ResponseWriter, status int, errorType string, message string, resource string, resourceID string) {
	writeJSON(w, status, map[string]any{
		"type":        errorType,
		"message":     message,
		"resource":    resource,
		"resource_id": resourceID,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("marshal test response: %v", err))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		panic(fmt.Sprintf("write test response: %v", err))
	}
}
