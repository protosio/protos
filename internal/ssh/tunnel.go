package ssh

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/util"
	"golang.org/x/crypto/ssh"
)

var log = util.GetLogger("ssh")

// Tunnel represents and SSH tunnel to a remote host
type Tunnel struct {
	sshHost   string
	sshUser   string
	sshAuth   ssh.AuthMethod
	sshConn   *ssh.Client
	listener  net.Listener
	localPort int
	target    string
	connMap   []chan bool
}

type forwarder struct {
	closed bool
	errsig chan bool
	close  chan bool
	lconn  net.Conn
	rconn  net.Conn
}

func (t *forwarder) pipe(src, dst net.Conn, name string) {
	buff := make([]byte, 0xffff)
	for {
		// read from the connection
		n, err := src.Read(buff)
		if err != nil {
			t.errSig(fmt.Sprintf("Read failed from '%s' -> '%s' (%s): ", src.RemoteAddr(), src.LocalAddr(), name), err)
			dst.Close()
			return
		}
		b := buff[:n]

		// write to the other connection
		n, err = dst.Write(b)
		if err != nil {
			t.errSig(fmt.Sprintf("Write failed to '%s' -> '%s' (%s): ", dst.LocalAddr(), dst.RemoteAddr(), name), err)
			src.Close()
			return
		}
	}
}

func (t *forwarder) errSig(s string, err error) {
	if t.closed {
		return
	}
	if err != io.EOF {
		log.Error(s, err)
	}
	t.errsig <- true
	t.closed = true
}

func (t *forwarder) proxy() {
	log.Debugf("Started forwarder for %p", t.lconn)
	go t.pipe(t.lconn, t.rconn, "outgoing")
	go t.pipe(t.rconn, t.lconn, "incoming")

	select {
	case <-t.errsig:
		log.Debugf("Forwarder %p closed because of underlying connections", t.lconn)
	case <-t.close:
		t.closed = true
		t.lconn.Close()
		t.rconn.Close()
		log.Debugf("Forwarder %p closed by user", t.lconn)
	}
}

func newForwarder(lconn, rconn net.Conn, close chan bool) *forwarder {
	return &forwarder{
		lconn:  lconn,
		rconn:  rconn,
		closed: false,
		errsig: make(chan bool),
		close:  close,
	}
}

// Start initiates the ssh tunnel
func (t *Tunnel) Start() (int, error) {
	// setup the local listener using a random port
	var err error
	t.listener, err = net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	t.localPort = t.listener.Addr().(*net.TCPAddr).Port

	// setup the SSH connection
	sshConfig := &ssh.ClientConfig{
		User: t.sshUser,
		Auth: []ssh.AuthMethod{t.sshAuth},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Always accept key.
			return nil
		}}
	t.sshConn, err = ssh.Dial("tcp", t.sshHost, sshConfig)
	if err != nil {
		return 0, err
	}

	// accept local connections and start the forwarding
	go func() {
		for {
			// accept a connection on localhost
			localConn, err := t.listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					log.Debug("Local SSH tunnel listener closed. Not accepting any new connections.")
					return
				}
				log.Errorf("Failed to accept connection via the SSH tunnel: %s", err)
				continue
			}

			// open a connection via the SSH connection, to the Protos backend
			remoteConn, err := t.sshConn.Dial("tcp", t.target)
			if err != nil {
				log.Errorf("Failed to establish remote connection (%s) over SSH tunnel (%s): %s", t.target, t.sshHost, err)
				return
			}

			close := make(chan bool, 1)
			forwarder := newForwarder(localConn, remoteConn, close)
			go forwarder.proxy()
			t.connMap = append(t.connMap, close)
		}
	}()

	return t.localPort, nil
}

// Close terminates the SSH tunnel
func (t *Tunnel) Close() error {
	// close the listener and the rest of the connections
	err := t.listener.Close()
	if err != nil {
		return errors.Wrap(err, "Error while closing local tunnel listener")
	}
	for _, close := range t.connMap {
		close <- true
	}
	err = t.sshConn.Close()
	if err != nil {
		return errors.Wrap(err, "Error while closing ssh tunnel connection")
	}

	return nil
}

// NewTunnel creates and returns an SSHTunnel
func NewTunnel(sshHost string, sshUser string, sshAuth ssh.AuthMethod, tunnelTarget string) *Tunnel {
	return &Tunnel{sshHost: sshHost, sshUser: sshUser, sshAuth: sshAuth, target: tunnelTarget}
}
