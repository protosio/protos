package types

// APIAuthPath is the url prefix for the auth api
const APIAuthPath string = "/api/v1/auth"

// APIExternalPath is the url prefix for the external api
const APIExternalPath string = "/api/v1/e"

// APIInternalPath is the url prefix for the internal api
const APIInternalPath string = "/api/v1/i"

// UserDevice - represents a device that a user uses to connect to the instances. A user can have multiple devices (laptop, mobile phone etc)
type UserDevice struct {
	Name      string `json:"name"`
	PublicKey string `json:"publickey"`
	Network   string `json:"network"` // Network should be specified in CIDR notation
}

// ReqInit - request payload for the registration endpoint
type ReqInit struct {
	Username        string       `json:"username"`
	Name            string       `json:"name"`
	Password        string       `json:"password"`
	ConfirmPassword string       `json:"confirmpassword"`
	Domain          string       `json:"domain"`
	Network         string       `json:"network"` // Network should be specified in CIDR notation
	Devices         []UserDevice `json:"devices"`
}

// RespInit - response payload for the registration endpoint
type RespInit struct {
	InstacePubKey string `json:"instancepubkey"`
	InstanceIP    string `json:"instanceip"`
}
