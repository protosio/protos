package client

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"

	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/pkg/types"
	"golang.org/x/net/publicsuffix"
)

// ExternalClient talks to the external Protos API
type ExternalClient interface {
	InitInstance(name string, network string, domain string, devices []auth.UserDevice) (net.IP, ed25519.PublicKey, error)
}

type externalClient struct {
	HTTPclient *http.Client
	apiPath    string
	host       string
	username   string
	password   string
}

// makeRequest prepares and sends a request to the protos backend
func (ec externalClient) makeRequest(method string, path string, body io.Reader) ([]byte, error) {
	urlStr := "http://" + ec.host + ec.apiPath + "/" + path
	errMsg := fmt.Sprintf("'%s' request '%s'", method, urlStr)

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
func (ec *externalClient) InitInstance(name string, network string, domain string, devices []auth.UserDevice) (net.IP, ed25519.PublicKey, error) {
	reqJSON, err := json.Marshal(types.ReqInit{
		Username:        ec.username,
		Password:        ec.password,
		ConfirmPassword: ec.password,
		Name:            name,
		Domain:          domain,
		Network:         network,
		Devices:         devices,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to init instance: %w", err)
	}

	// register the user
	ec.apiPath = types.APIAuthPath
	httpResp, err := ec.makeRequest(http.MethodPost, "init", bytes.NewBuffer(reqJSON))
	ec.apiPath = types.APIExternalPath
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to init instance: %w", err)
	}

	// decode response
	initResp := types.RespInit{}
	err = json.Unmarshal(httpResp, &initResp)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode http message from Protos: %w", err)
	}

	// prepare IP and public key of instance
	ip := net.ParseIP(initResp.InstanceIP)
	if ip == nil {
		return nil, nil, fmt.Errorf("Failed to parse IP: %w", err)
	}
	var pubKey ed25519.PublicKey
	pubKey, err = base64.StdEncoding.DecodeString(initResp.InstacePubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode public key: %w", err)
	}

	// stop the init endpoint
	_, err = ec.makeRequest(http.MethodGet, "init/finish", bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to init instance: %w", err)
	}

	return ip, pubKey, err
}

// NewInitClient creates and returns a client that's used to initialize an instance
func NewInitClient(host string, username string, password string) ExternalClient {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		log.Fatal(err)
	}

	ec := &externalClient{
		host:       host,
		username:   username,
		password:   password,
		HTTPclient: &http.Client{Jar: jar},
		apiPath:    types.APIExternalPath,
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
		apiPath:    types.APIExternalPath,
	}

	return ec, nil
}
