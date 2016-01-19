package daemon

import (
	"encoding/json"
	"fmt"
	"github.com/apexskier/httpauth"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"os"
	"strconv"
)

var (
	backend     httpauth.GobFileAuthBackend
	aaa         httpauth.Authorizer
	roles       map[string]httpauth.Role
	port        = 8009
	backendfile = "auth.gob"
)

func Websrv() {
	var err error
	// create the backend storage, remove when all done
	os.Create(backendfile)
	defer os.Remove(backendfile)

	// create the backend
	backend, err = httpauth.NewGobFileAuthBackend(backendfile)
	if err != nil {
		panic(err)
	}

	// create some default roles
	roles = make(map[string]httpauth.Role)
	roles["user"] = 30
	roles["admin"] = 80
	aaa, err = httpauth.NewAuthorizer(backend, []byte("cookie-encryption-key"), "user", roles)

	// create a default user
	hash, err := bcrypt.GenerateFromPassword([]byte("adminadmin"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	defaultUser := httpauth.UserData{Username: "admin", Email: "admin@localhost", Hash: hash, Role: "admin"}
	err = backend.SaveUser(defaultUser)
	if err != nil {
		panic(err)
	}

	rtr := mux.NewRouter()

	//fileHandler := http.StripPrefix("/static/", http.FileServer(http.Dir(Gconfig.StaticAssets)))
	fileHandler := http.FileServer(http.Dir(Gconfig.StaticAssets))

	//rtr.HandleFunc("/", IndexHandler)
	rtr.HandleFunc("/login", getLogin).Methods("GET")
	rtr.HandleFunc("/login", postLogin).Methods("POST")
	rtr.HandleFunc("/logout", handleLogout)
	rtr.HandleFunc("/apps", AppsHandler)
	rtr.HandleFunc("/apps/{app}", AppHandler)
	rtr.HandleFunc("/store", StoreHandler)
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

	log.Debug("Received app state change request")
	vars := mux.Vars(r)

	decoder := json.NewDecoder(r.Body)
	var app_params App
	err := decoder.Decode(&app_params)
	if err != nil {
		log.Error("Invalid request: ", r.Body)
	}

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

func StoreHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		log.Println(r.FormValue("download"))
		DownloadApp(r.FormValue("download"))
	}

	apps := SearchApps()
	log.Println(apps)

	data := struct {
		Title string
		Apps  []AppSearch
	}{
		"Apps",
		apps,
	}

	t := template.Must(template.ParseFiles("templates/store.html", "templates/head.html", "templates/navbar.html"))
	t.Execute(w, data)

}

func getLogin(w http.ResponseWriter, r *http.Request) {
	//messages := aaa.Messages(w, r)

	if user, err := aaa.CurrentUser(w, r); err == nil {
		log.Println("User", user.Username, "already logged in")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

	data := struct {
		Title string
	}{
		"Login",
	}

	t := template.Must(template.ParseFiles("templates/login.html", "templates/head.html"))
	t.Execute(w, data)

}

func postLogin(rw http.ResponseWriter, req *http.Request) {
	log.Println(req)
	username := req.PostFormValue("username")
	password := req.PostFormValue("password")
	log.Println(username)
	log.Println(password)
	if err := aaa.Login(rw, req, username, password, "/"); err != nil && err.Error() == "httpauth: already authenticated" {
		log.Println("User already logged in")
		http.Redirect(rw, req, "/", http.StatusSeeOther)
	} else if err != nil {
		log.Println("Error: ", err.Error())
		http.Redirect(rw, req, "/login", http.StatusSeeOther)
	}
}

func handleLogout(rw http.ResponseWriter, req *http.Request) {
	if err := aaa.Logout(rw, req); err != nil {
		fmt.Println(err)
		// this shouldn't happen
		return
	}
	log.Println("User logged out successfuly")
	http.Redirect(rw, req, "/", http.StatusSeeOther)
}
