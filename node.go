package orchestrator

import (
	"errors"
	"fmt"
	"io/ioutil"

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
	key, err := ioutil.ReadFile(n.Connection.SSHKey)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: n.Connection.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
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
		if n.Connection.Port == "" {
			n.Connection.Port = "22"
		}
		if n.Connection.User == "" {
			return errors.New("Node validation: undefined User")
		}
		if n.Connection.SSHKey == "" {
			return errors.New("Node validation: undefined SSHKey")
		}
	}
	return nil
}
