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

	rtr.HandleFunc("/", IndexHandler)
	rtr.HandleFunc("/apps", AppsHandler)
	rtr.HandleFunc("/apps/{app}", AppHandler)
	rtr.PathPrefix("/static").Handler(fileHandler)
	http.Handle("/", rtr)

	log.Println("Listening...")
	http.ListenAndServe(":9000", nil)

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	apps := GetApps()
	app_count := len(apps)

	data := struct {
		Title    string
		AppCount int
	}{
		"Dashboard",
		app_count,
	}

	t := template.Must(template.ParseFiles("templates/index.html", "templates/head.html", "templates/navbar.html"))
	t.Execute(w, data)

}

func AppsHandler(w http.ResponseWriter, r *http.Request) {

	apps := GetApps()

	data := struct {
		Title string
		Apps  map[string]*App
	}{
		"Apps",
		apps,
	}

	t := template.Must(template.ParseFiles("templates/apps.html", "templates/head.html", "templates/navbar.html"))
	t.Execute(w, data)

}

func AppHandler(w http.ResponseWriter, r *http.Request) {

	//apps := GetApps()

	data := struct {
		Title string
		Apps  string
	}{
		"Apps",
		"yolo",
	}

	t := template.Must(template.ParseFiles("templates/app.html", "templates/head.html", "templates/navbar.html"))
	t.Execute(w, data)

}
