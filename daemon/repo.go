package daemon

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type AppSearch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func SearchApps() []AppSearch {
	response, err := http.Get("http://dexter.giurgiu.io:5000/v1/search")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Println(string(contents))

	var objmap map[string]*json.RawMessage
	err = json.Unmarshal(contents, &objmap)

	var searchResult []AppSearch
	err = json.Unmarshal(*objmap["results"], &searchResult)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	return searchResult
}
