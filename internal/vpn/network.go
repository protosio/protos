package vpn

// import (
// 	"net"

// 	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
// )

// type Address struct {
// 	net.IPNet
// 	Peer *net.IPNet
// }

// type Route struct {
// 	Dest net.IPNet
// 	Src  net.IP
// }

// type Link interface {
// 	Interface() net.Interface
// 	Name() string
// 	Index() int

// 	IsUp() bool
// 	SetUp(bool) error
// 	Addrs() ([]Address, error)
// 	DelAddr(a Address) error
// 	AddAddr(a Address) error

// 	ConfigureWG(wgtypes.Config) error
// 	WGConfig() (*wgtypes.Device, error)

// 	AddRoute(Route) error
// 	DelRoute(Route) error
// }

// // Manager deals with the management of network interfaces
// type Manager interface {
// 	// Links returns all the interfaces managed by the manager
// 	Links() ([]Link, error)
// 	// CreateLink creates an interface and returns it
// 	CreateLink(name string) (Link, error)
// 	// DelLink deletes an interface based on the provided interface number
// 	DelLink(name string) error
// 	// GetLink returns a link
// 	GetLink(name string) (Link, error)

// 	Close() error
// }
