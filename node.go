package orchestartor

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ssh "github.com/melbahja/goph"
	shell "github.com/progrium/go-shell"
)

type Node struct {
	Key              NodeKey      // in a case of 0, Key will be declared as rand.Int()
	StartImmediately bool         // starts node immediately
	Romote           bool         // Local/Remote
	HTTPAccess       []HTTPAccess // http access settings
	OS               string       // linux / darwin / windows
	Connection       Connection
	Commands         Commands
	Settings         Settings
}

type Connection struct {
	Address    string
	User       string
	SSHKey     string // SSHKey is a path to private key (client key)
	PassPhrase string
}

// HTTPAccess smth like in consul config
type HTTPAccess struct {
	Method     string
	Address    string
	StatusCode int
}

type Commands struct {
	Start, Stop, Restart, Reload, Status, IsActive string
}

type Settings struct {
	Restart string // default = never
	Reload  string // default = never
	Timeout int    // default = 300 seconds
	Closed  bool
}

// ISvc is an interface
type ISvc interface {
}

// NodeName is a {ServiceName}_{NodeKey}
type NodeName string

// NodeKey is a unique value for some service; can be 0 in configuration file
type NodeKey int

// func (a *Node) GetISvc() (interface{ ISvc }, error) {
// 	switch a.OS {
// 	case "linux":
// 		return &SystemD{}, nil
// 	case "darwin":
// 		return &LaunchD{}, nil
// 	case "windows":
// 		return &WindowsService{}, nil
// 	default:
// 		return nil, errors.New("Unknown OS")
// 	}
// }

func (n *Node) Start() (string, error) {
	return n.run(n.Commands.Start)
}

func (n *Node) Stop() (string, error) {
	return n.run(n.Commands.Stop)
}

func (n *Node) Restart() (string, error) {
	return n.run(n.Commands.Restart)
}

func (n *Node) Reload() (string, error) {
	return n.run(n.Commands.Reload)
}

func (n *Node) IsActive() (string, error) {
	return n.run(n.Commands.IsActive)
}

func (n *Node) Status() *Status {
	for _, aMethod := range n.HTTPAccess {
		err := aMethod.do()
		if err != nil {
			return &Status{false, StatusHTTPAccesMethodFailed, err}
		}
	}
	if n.Commands.Status != "" {
		_, err := n.run(n.Commands.Status)
		if err != nil {
			return &Status{false, StatusGetStatusFailed, err}
		}
	}
	if n.Commands.IsActive != "" {
		out, err := n.run(n.Commands.IsActive)
		if err != nil {
			return &Status{false, StatusGetIsActiveFailed, err}
		}
		if out == "inactive" {
			return &Status{false, StatusStopped, nil}
		}
	}
	return &Status{true, StatusRunning, nil}
}

func (n *Node) Go(srvName ServiceName, event chan NodeStatus) error {
	if srvName == OrchestratorServiceName {
		time.Sleep(time.Duration(3) * time.Second)
	}
	for {
		if n.Settings.Closed {
			event <- NodeStatus{NewNodeName(srvName, n.Key), n, nil, time.Now().String(), ""}
			break
		}
		d := time.Duration(n.Settings.Timeout) * time.Second
		event <- NodeStatus{NewNodeName(srvName, n.Key), n, n.Status(), time.Now().String(), time.Now().Add(d).String()}
		time.Sleep(d)
	}
	return nil
}

func (n *Node) run(command string) (str string, err error) {
	if n.Romote {
		client, err := ssh.New(n.Connection.User, n.Connection.Address, ssh.Key(n.Connection.SSHKey, n.Connection.PassPhrase))
		if err != nil {
			return "", err
		}
		// defer client.Close()
		out, err := client.Run(command)
		if err != nil {
			return "", err
		}
		str = string(out)
		err = client.Close()
		if err != nil {
			return "", err
		}
	} else {
		sh := shell.Run
		str = sh(command).Stdout.String()
	}
	return
}

func (n *Node) valid() bool {
	if n.Connection.User == "" {
		n.Connection.User = "root"
	}
	if n.Key == 0 {
		n.Key = NodeKey(rand.Int())
	}
	for _, httpAccess := range n.HTTPAccess {
		if !httpAccess.valid() {
			return false
		}
	}
	if _, ok := SettingsMap[n.Settings.Restart]; !ok {
		n.Settings.Restart = "never"
	}
	if _, ok := SettingsMap[n.Settings.Reload]; !ok {
		n.Settings.Reload = "never"
	}
	if n.Settings.Timeout <= 0 {
		n.Settings.Timeout = -1
		n.Settings.Closed = true
	}
	return true
}

func (name *NodeName) Parse() (ServiceName, NodeKey, error) {
	arr, err := strings.Split(string(*name), "_"), errors.New("Can't parse NodeName")
	if len(arr) != 2 {
		return "", 0, err
	}
	key, err := strconv.Atoi(arr[1])
	if err != nil {
		return "", 0, err
	}
	return ServiceName(arr[0]), NodeKey(key), nil
}

func NewNodeName(sName ServiceName, nKey NodeKey) NodeName {
	return NodeName(fmt.Sprintf("%s_%d", sName, nKey))
}

func (h *HTTPAccess) valid() bool {
	_, ok := HttpMethodMap[h.Method]
	if !ok {
		return false
	}
	_, err := url.ParseRequestURI(h.Address)
	if err != nil {
		return false
	}
	if h.StatusCode < 100 || h.StatusCode > 526 {
		return false
	}
	return true
}

func (h *HTTPAccess) do() error {
	request, err := http.NewRequest(h.Method, h.Address, strings.NewReader(""))
	if err != nil {
		return fmt.Errorf("HTTP access method: %s", err.Error())
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("HTTP access method: %s", err.Error())
	}
	if resp.StatusCode != h.StatusCode {
		return fmt.Errorf("HTTP access method: expected status code %d, got %d", h.StatusCode, resp.StatusCode)
	}
	return nil
}
