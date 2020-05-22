package core

type VPN interface {
	Start() error
	Stop() error
}
