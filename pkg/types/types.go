package types

// APIAuthPath is the url prefix for the auth api
const APIAuthPath string = "/api/v1/auth"

// APIExternalPath is the url prefix for the external api
const APIExternalPath string = "/api/v1/e"

// APIInternalPath is the url prefix for the internal api
const APIInternalPath string = "/api/v1/i"

// UserDevice - represents a device that a user uses to connect to the instances. A user can have multiple devices (laptop, mobile phone etc)
type UserDevice struct {
	Name      string `json:"name" validate:"required"`
	PublicKey string `json:"publickey" validate:"base64"` // ed25519 base64 encoded public key
	Network   string `json:"network" validate:"cidrv4"`   // CIDR notation
}

// ReqInit - request payload for the registration endpoint
type ReqInit struct {
	Username        string       `json:"username" validate:"required"`
	Name            string       `json:"name" validate:"required"`
	Password        string       `json:"password" validate:"min=10,max=100"`
	ConfirmPassword string       `json:"confirmpassword" validate:"eqfield=Password"`
	Domain          string       `json:"domain" validate:"fqdn"`
	Network         string       `json:"network" validate:"cidrv4"` // CIDR notation
	Devices         []UserDevice `json:"devices" validate:"gt=0,dive"`
}

// RespInit - response payload for the registration endpoint
type RespInit struct {
	InstacePubKey string `json:"instancepubkey" validate:"base64"` // ed25519 base64 encoded public key
	InstanceIP    string `json:"instanceip" validate:"ipv4"`
}
