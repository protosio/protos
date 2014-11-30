package daemon

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
)

type Page struct {
	Title string
	Body  []byte
}

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

	p := &Page{Title: "Egor dashboard", Body: []byte("page content")}

	t := template.Must(template.ParseFiles("templates/index.html", "templates/head.html", "templates/navbar.html"))
	t.Execute(w, p)

}
