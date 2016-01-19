package daemon

import (
	"bytes"
	"encoding/json"
	"github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type AppSearch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func DownloadApp(name string) {

	client := Gconfig.DockerClient
	var buf bytes.Buffer

	test := strings.Split(name, "/")
	log.Info("Downloading [", test[1], "]")

	opts := docker.PullImageOptions{
		Repository:   "dexter.giurgiu.io:5000/" + test[1],
		Registry:     "dexter.giurgiu.io:5000",
		OutputStream: &buf,
	}
	err := client.PullImage(opts, docker.AuthConfiguration{})
	if err != nil {
		log.Warn(err)
	}

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
	//log.Println(string(contents))

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
