package pcrypto

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// ExecuteCommand opens a session using the provided client and executes the provided command
func ExecuteCommand(cmd string, client *ssh.Client) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", errors.Wrap(err, "Failed to create new sessions")
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		session.Close()
		return "", errors.Wrap(err, "Request for pseudo terminal failed")
	}

	log.Debugf("Executing (SSH) command '%s'", cmd)
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), errors.Wrapf(err, "Failed to execute command '%s'", cmd)
	}

	session.Close()

	return string(output), nil

}

func NewConnection(host string, user string, auth ssh.AuthMethod, maxRetries int) (*ssh.Client, error) {
	hostKeyCallback, err := TrustOnFirstUseHostKeyCallback()
	if err != nil {
		return nil, err
	}
	return NewConnectionWithHostKeyCallback(host, user, auth, maxRetries, hostKeyCallback)
}

func NewConnectionWithHostKeyCallback(host string, user string, auth ssh.AuthMethod, maxRetries int, hostKeyCallback ssh.HostKeyCallback) (*ssh.Client, error) {
	if user == "" {
		user = "root"
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	if hostKeyCallback == nil {
		return nil, errors.New("SSH host key callback is required")
	}
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			auth,
		},
		HostKeyCallback: hostKeyCallback,
	}
	address := sshAddress(host)

	tries := 0
	var client *ssh.Client
	var err error
	for {
		tries++
		if tries > maxRetries {
			return nil, errors.Wrapf(err, "Failed to open SSH connection to '%s@%s'", user, address)
		}
		client, err = ssh.Dial("tcp", address, sshConfig)
		if err != nil {
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	return client, nil
}

func sshAddress(host string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, "22")
}

func TrustOnFirstUseHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "find home directory for SSH known_hosts")
	}
	path := filepath.Join(home, ".ssh", "known_hosts")
	return knownHostsFileCallback(path), nil
}

func EphemeralTrustOnFirstUseHostKeyCallback() ssh.HostKeyCallback {
	var mu sync.Mutex
	seen := map[string][]byte{}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		host := hostname
		if host == "" && remote != nil {
			host = remote.String()
		}
		if host == "" {
			return errors.New("SSH host is empty")
		}
		keyBytes := key.Marshal()

		mu.Lock()
		defer mu.Unlock()

		known, found := seen[host]
		if !found {
			seen[host] = append([]byte(nil), keyBytes...)
			return nil
		}
		if !bytes.Equal(known, keyBytes) {
			return fmt.Errorf("SSH host key changed for %s during this connection attempt", host)
		}
		return nil
	}
}

func knownHostsFileCallback(path string) ssh.HostKeyCallback {
	var mu sync.Mutex
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		mu.Lock()
		defer mu.Unlock()

		callback, err := knownhosts.New(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("load SSH known_hosts: %w", err)
		}
		if err == nil {
			if verifyErr := callback(hostname, remote, key); verifyErr == nil {
				return nil
			} else {
				var keyErr *knownhosts.KeyError
				if !stderrors.As(verifyErr, &keyErr) || len(keyErr.Want) > 0 {
					return verifyErr
				}
			}
		}

		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return fmt.Errorf("create SSH known_hosts directory: %w", err)
		}
		line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("open SSH known_hosts: %w", err)
		}
		defer file.Close()
		if _, err := fmt.Fprintln(file, line); err != nil {
			return fmt.Errorf("record SSH host key: %w", err)
		}
		return nil
	}
}
