package daemon

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
)

func Websrv() {
	rtr := mux.NewRouter()

	fileHandler := http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))

	rtr.HandleFunc("/", MainHandler)
	rtr.PathPrefix("/static").Handler(fileHandler)
	http.Handle("/", rtr)

	log.Println("Listening...")
	http.ListenAndServe(":9000", nil)

}

func MainHandler(w http.ResponseWriter, r *http.Request) {

	apps := GetApps()
	app_count := len(apps)

	data := struct {
		Title    string
		AppCount int
	}{
		"Egor dashboard",
		app_count,
	}

	t := template.Must(template.ParseFiles("templates/index.html", "templates/head.html", "templates/navbar.html"))
	t.Execute(w, data)

}
