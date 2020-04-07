package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"

	"github.com/protosio/protos/pkg/types"
	"golang.org/x/net/publicsuffix"
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

	errMsg := fmt.Sprintf("'%s' request '%s'", method, path)

	urlStr := "http://" + ec.host + "/" + ec.apiPath + "/" + path
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return []byte{}, fmt.Errorf("%v: %v", errMsg, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ec.HTTPclient.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("%v: %v", errMsg, err)
	}
	defer resp.Body.Close()

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("%v: %v", errMsg, err)
	}

	if resp.StatusCode != 200 {
		httperr := httpErr{}
		err := json.Unmarshal(payload, &httperr)
		if err != nil {
			return []byte{}, fmt.Errorf("Failed to decode error message from Protos: %v", err)
		}
		return []byte{}, fmt.Errorf("%v: %v", errMsg, errors.New(httperr.Error))
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
	ec.apiPath = "api/v1/auth"
	_, err = ec.makeRequest(http.MethodPost, "register", bytes.NewBuffer(reqJSON))
	ec.apiPath = "api/v1/e"
	if err != nil {
		return fmt.Errorf("Failed to init instance: %v", err)
	}

	// stop the init endpoint
	_, err = ec.makeRequest(http.MethodGet, "init/finish", bytes.NewBuffer(reqJSON))
	if err != nil {
		return fmt.Errorf("Failed to init instance: %v", err)
	}

	return err
}

// NewInitClient creates and returns a client that's used to initialize an instance
func NewInitClient(host string, username string, password string, domain string) ExternalClient {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}
	ec := &externalClient{
		host:       host,
		username:   username,
		password:   password,
		domain:     domain,
		HTTPclient: &http.Client{Jar: jar},
		apiPath:    "api/v1/e",
	}
	return ec
}

// NewExternalClient creates and returns a client for the external Protos API
func NewExternalClient(host string, username string, password string) (ExternalClient, error) {
	ec := &externalClient{
		host:       host,
		username:   username,
		password:   password,
		HTTPclient: &http.Client{},
		apiPath:    "api/v1/e",
	}

	return ec, nil
}
