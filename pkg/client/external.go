package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/protosio/protos/pkg/types"
)

// ExternalClient talks to the external Protos API
type ExternalClient interface {
	InitInstance() error
}

type externalClient struct {
	HTTPclient *http.Client
	apiPath    string
	host       string
	username   string
	password   string
	domain     string
}

// makeRequest prepares and sends a request to the protos backend
func (ec externalClient) makeRequest(method string, path string, body io.Reader) ([]byte, error) {

	url := "http://" + ec.host + "/" + ec.apiPath + "/" + path
	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		return []byte{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ec.HTTPclient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	if resp.StatusCode != 200 {
		httperr := httpErr{}
		err := json.Unmarshal(payload, &httperr)
		if err != nil {
			return []byte{}, fmt.Errorf("Failed to decode error message from Protos: %s", err.Error())
		}
		return []byte{}, errors.New(httperr.Error)
	}

	return payload, nil
}

// InitInstance initializes a newly deployed Protos instance
func (ec *externalClient) InitInstance() error {
	reqJSON, err := json.Marshal(types.ReqRegister{
		Username:        ec.username,
		Password:        ec.password,
		ConfirmPassword: ec.password,
		Domain:          ec.domain,
	})
	if err != nil {
		return fmt.Errorf("Failed to init instance: %v", err)
	}

	// register the user
	ec.apiPath = "/api/v1/auth"
	_, err = ec.makeRequest(http.MethodGet, "/register", bytes.NewBuffer(reqJSON))
	ec.apiPath = "api/v1/e"
	if err != nil {
		return fmt.Errorf("Failed to init instance: %v", err)
	}

	// stop the init endpoint
	_, err = ec.makeRequest(http.MethodGet, "/init/finish", bytes.NewBuffer(reqJSON))
	if err != nil {
		return fmt.Errorf("Failed to init instance: %v", err)
	}

	return err
}

// NewExternalClient creates and returns a client for the external Protos API
func NewExternalClient(host string, username string, password string, domain string) ExternalClient {
	ec := &externalClient{
		host:       host,
		username:   username,
		password:   password,
		domain:     domain,
		HTTPclient: &http.Client{},
		apiPath:    "api/v1/e",
	}

	return ec
}
