package daemon

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func Websrv() {
	rtr := mux.NewRouter()

	//rtr.HandleFunc("/admin", webadmin).Methods("GET")
	//http.Handle("/", rtr)

	rtr.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	http.Handle("/", rtr)

	log.Println("Listening...")
	http.ListenAndServe(":9000", nil)

}
