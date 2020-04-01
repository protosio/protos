package types

// ReqRegister - request payload for the registration endpoint
type ReqRegister struct {
	Username        string `json:"username"`
	Name            string `json:"name"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmpassword"`
	Domain          string `json:"domain"`
}

// RespRegister - response payload for the registration endpoint
type RespRegister struct {
	Username string `json:"username"`
}
