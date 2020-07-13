package orchestrator

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	ssh "github.com/melbahja/goph"
)

type Node struct {
	*NodeConfiguration
	isConnected bool        // is connection ok
	client      *ssh.Client // ssh client
}

type NodeInfo struct {
	Connected bool
	*NodeConfiguration
}

type NodeInfoResponse struct {
	IsConnected bool
	Error       string
}

type NodeConfiguration struct {
	NodeName         string
	OS               string // linux / darwin / windows
	StartImmediately bool   // starts node immediately
	Remote           bool   // Local / Remote
	Connection       *Connection
}

type Connection struct {
	Host       string
	Port       string
	User       string
	SSHKey     string // SSHKey is a path to private key (client key)
	PassPhrase string
}

func NewNode(config *NodeConfiguration) (*Node, error) {
	node := &Node{config, false, nil}
	if err := node.Valid(); err != nil {
		return nil, err
	}
	return node, nil
}

func (n *Node) Connect() (err error) {
	if n.Remote {
		server := fmt.Sprintf("%s:%s", n.Connection.Host, n.Connection.Port)
		n.client, err = ssh.New(n.Connection.User, server, ssh.Key(n.Connection.SSHKey, n.Connection.PassPhrase))
		if err != nil {
			return err
		}
		return
	}
	return fmt.Errorf("%s node is configured as local", n.NodeName)
}

func (n *Node) Disconnect() error {
	if n.Remote {
		return n.client.Close()
	}
	return fmt.Errorf("%s node is configured as local", n.NodeName)
}

func (n *Node) IsConnected() (bool, error) {
	if n.Remote {
		if n.client == nil {
			n.isConnected = false
			return false, fmt.Errorf("%s node has nil Connection", n.NodeName)
		}
		session, err := n.client.NewSession()
		if err != nil {
			n.isConnected = false
			return false, err
		}
		err = session.Close()
		if err != nil {
			n.isConnected = false
			return false, err
		}
	}
	n.isConnected = true
	return true, nil
}

func (n *Node) ServiceStatus(srvName string) error {
	switch n.OS {
	case OSDarwin: // only local
		command := fmt.Sprintf(DarwinTryIsActiveFormatString, srvName)
		out, err := n.runcommand(srvName, command)
		if err != nil {
			return err
		}
		if !strings.Contains(out, "0") {
			return fmt.Errorf("%s", StatusInactive)
		}
	case OSLinux: // local + remote
		command := fmt.Sprintf(LinuxTryIsActiveFormatString, srvName)
		out, err := n.runcommand(srvName, command)
		if err != nil {
			return err
		}
		if !strings.Contains(out, "0") {
			return fmt.Errorf("%s", StatusInactive)
		}
	default:
		return fmt.Errorf("Node error: unknown OS %s", n.OS)
	}
	return nil
}

func (n *Node) StartService(srvName string) error {
	command := ""
	switch n.OS {
	case OSDarwin: // only local
		command = fmt.Sprintf(DarwinStartServiceFormatString, srvName)
	case OSLinux: // local + remote
		command = fmt.Sprintf(LinuxStartServiceFormatString, srvName)
	}
	_, err := n.runcommand(srvName, command)
	return err
}

func (n *Node) StopService(srvName string) error {
	command := ""
	switch n.OS {
	case OSDarwin: // only local
		command = fmt.Sprintf(DarwinStopServiceFormatString, srvName)
	case OSLinux: // local + remote
		command = fmt.Sprintf(LinuxStopServiceFormatString, srvName)
	}
	_, err := n.runcommand(srvName, command)
	return err
}

func (n *Node) runcommand(srvName, command string) (string, error) {
	var out []byte
	_, err := n.IsConnected()
	if err != nil {
		return "", err
	}
	switch n.OS {
	case OSDarwin: // only local
		if n.Remote {
			return "", fmt.Errorf("Node error: remote access for OS %s is not provided", n.OS)
		} else {
			out, err = exec.Command("bash", "-c", command).Output()
			if err != nil {
				return "", err
			}
		}
	case OSWindows: // no local, no remote
		if n.Remote {
			return "", fmt.Errorf("Node error: remote access for OS %s is not provided", n.OS)
		} else {
			return "", fmt.Errorf("Node error: local access for OS %s is not provided", n.OS)
		}
	case OSLinux: // local + remote
		if n.Remote {
			out, err = n.client.Run(command)
			if err != nil {
				return "", err
			}
		} else {
			out, err = exec.Command("bash", "-c", command).Output()
			if err != nil {
				return "", err
			}
		}
	default:
		return "", fmt.Errorf("Node error: unknown OS %s", n.OS)
	}
	return string(out), nil
}

func (n *Node) Valid() error {
	if n.NodeName == "" {
		return errors.New("Node validation: undefined Name")
	}
	if n.OS != OSDarwin && n.OS != OSLinux && n.OS != OSWindows {
		return errors.New("Node validation: unknown OS")
	}
	if n.Remote {
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

func getThisNodeConfiguration() *NodeConfiguration {
	return &NodeConfiguration{
		NodeName:         NameOfThisNode,
		OS:               os.Getenv("GOOS"),
		StartImmediately: true,
		Remote:           false,
	}
}
