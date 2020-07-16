package orchestrator

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"

	"golang.org/x/crypto/ssh"
)

type Node struct {
	NodeStatus int
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

func NewNode(config *NodeConfiguration) (*Node, error) {
	node := &Node{StatusInitialized, *config}
	if err := node.Valid(); err != nil {
		return nil, err
	}
	return node, nil
}

func (n *Node) Connect() (*ssh.Client, error) {
	if n.Connection == nil {
		return nil, fmt.Errorf("Node access: %s node is configured as local", n.NodeName)
	}
	auth := make([]ssh.AuthMethod, 0)
	key, err := ioutil.ReadFile(n.Connection.SSHKey)
	if err != nil {
		return nil, err
	}
	if n.Connection.PassPhrase == "" {
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(n.Connection.PassPhrase))
		if err != nil {
			return nil, err
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	config := &ssh.ClientConfig{
		User: n.Connection.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	server := n.Connection.Host + ":" + n.Connection.Port
	client, err := ssh.Dial("tcp", server, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (n *Node) Valid() error {
	if n.NodeName == "" {
		return errors.New("Node validation: undefined Name")
	}
	if n.OS != OSDarwin && n.OS != OSLinux && n.OS != OSWindows {
		return errors.New("Node validation: unknown OS")
	}
	if n.Connection != nil {
		if n.Connection == nil {
			return errors.New("Node validation: nil Connection for remote node")
		}
		if n.Connection.Host == "" {
			return errors.New("Node validation: undefined Host")
		}
		if n.Connection.SSHKey == "" {
			return errors.New("Node validation: undefined SSHKey path")
		}
		if n.Connection.Port == "" {
			n.Connection.Port = "22"
		}
		if n.Connection.User == "" {
			n.Connection.User = "root"
		}
	}
	return nil
}
