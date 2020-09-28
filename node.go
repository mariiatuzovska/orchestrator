package orchestrator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Node struct {
	nodeStatus int
	NodeConfiguration
}

type NodeConfiguration struct {
	NodeName   string
	OS         string // linux / darwin / windows
	Connection *Connection
}

type Connection struct {
	Host       string
	Port       string
	User       string
	SSHKey     string // SSHKey is a path to private key (client key)
	PassPhrase string
}

func NewNode(config *NodeConfiguration) *Node {
	return &Node{StatusInitialized, *config}
}

func (n *Node) Status() int {
	return n.nodeStatus
}

func (n *Node) Connect() (*ssh.Client, error) {
	if n.Connection == nil {
		return nil, fmt.Errorf("Node access: %s node is configured as local", n.NodeName)
	}
	return n.Connection.connect()
}

func (n *Node) Valid() error {
	if n.NodeName == "" {
		return errors.New("Node validation: undefined Name")
	}
	if n.OS != OSDarwin && n.OS != OSLinux {
		return errors.New("Node validation: unknown OS")
	}
	if n.Connection != nil {
		return n.Connection.Valid()
	}
	return nil
}

func (c *Connection) connect() (*ssh.Client, error) {
	auth := make([]ssh.AuthMethod, 0)
	key, err := ioutil.ReadFile(c.SSHKey)
	if err != nil {
		return nil, err
	}
	if c.PassPhrase == "" {
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(c.PassPhrase))
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	config := &ssh.ClientConfig{
		User: c.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	client, err := ssh.Dial("tcp", c.Host+":"+c.Port, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (c *Connection) Valid() error {
	if c == nil {
		return errors.New("Connection validation: nil Connection")
	}
	if c.Host == "" {
		return errors.New("Connection validation: undefined Host")
	}
	if c.SSHKey == "" {
		return errors.New("Connection validation: undefined SSHKey path")
	} else if strings.Contains(c.SSHKey, "~/") { // linux + darwin
		out, err := exec.Command("bash", "-c", "echo ~").Output()
		if err != nil {
			return err
		}
		if out[len(out)-1] == 10 { // \n
			out = out[:len(out)-1]
		}
		c.SSHKey = path.Join(string(out), strings.Replace(c.SSHKey, "~/", "", 1))
	}
	if c.Port == "" {
		c.Port = "22"
	}
	if c.User == "" {
		c.User = "root"
	}
	return nil
}
