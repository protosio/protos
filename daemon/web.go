package daemon

import (
	"encoding/json"
	"github.com/abbot/go-http-auth"
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
	"strconv"
)

func Secret(user, realm string) string {
	if user == "alex" {
		// password is "hello"
		return "$1$dlPL2MqE$oQmn16q49SqdmhenQuNgs1"
	}
	return ""
}

func Websrv() {

	rtr := mux.NewRouter()
	authenticator := auth.NewBasicAuthenticator("localhost", Secret)

	fileHandler := http.FileServer(http.Dir(Gconfig.StaticAssets))

	rtr.HandleFunc("/apps", auth.JustCheck(authenticator, AppsHandler))
	rtr.HandleFunc("/apps/{app}", AppHandler)
	rtr.PathPrefix("/static").Handler(fileHandler)
	rtr.PathPrefix("/").Handler(fileHandler)
	http.Handle("/", rtr)

	port := strconv.Itoa(Gconfig.Port)
	log.Println("Listening on port " + port)
	http.ListenAndServe(":"+port, nil)

}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	//apps := GetApps()
	//app_count := len(apps)

	data := struct {
		Title string
	}{
		"Dashboard",
	}

	//t := template.Must(template.ParseFiles("templates/index.html", "templates/head.html", "templates/navbar.html"))
	t := template.Must(template.ParseFiles("templates/index_cljs.html"))
	t.Execute(w, data)

}

func AppsHandler(w http.ResponseWriter, r *http.Request) {

	apps := GetApps()

	data := struct {
		Apps map[string]*App
	}{
		apps,
	}

	//t := template.Must(template.ParseFiles("templates/apps.html", "templates/head.html", "templates/navbar.html"))
	//t.Execute(w, data)
	log.Println("Sending response: ", apps)
	json.NewEncoder(w).Encode(data)

}

func AppHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	decoder := json.NewDecoder(r.Body)
	var app_params App
	err := decoder.Decode(&app_params)
	if err != nil {
		log.Error("Invalid request: ", r.Body)
	}
	log.Debug("Received app state change request: ", app_params)

	appname := vars["app"]

	app := GetApp(appname)

	if r.Method == "POST" {
		if app_params.Status.Running == true {
			app.Start()
		} else if app_params.Status.Running == false {
			app.Stop()
		}
	}

	log.Println("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}
