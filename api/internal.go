package api

import "net/http"

var internalRoutes = routes{
	route{
		"registerResource",
		"POST",
		"/internal/resource",
		registerResource,
	},
}

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func registerResource(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
