package orchestrator

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	ssh "github.com/melbahja/goph"
)

type Node struct {
	Alive            bool   // is node alive
	StartImmediately bool   // starts node immediately
	Romote           bool   // Local / Remote
	OS               string // linux / darwin / windows
	Connection       *Connection
	Commands         Commands
	HTTPAccess       []*HTTPAccess // http access settings
	Settings         *Settings
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
	Headers    map[string]string
}

type Commands map[CommandName]*Command

type CommandName string

type Command struct {
	GetStatus bool
	Stdin     string
	Stdout    string
}

type SettingValue string

type Settings struct {
	Restart SettingValue // default = never
	Reload  SettingValue // default = never
	Timeout int
}

// NodeName is a {ServiceName}_{NodeKey}
type NodeName string

type NodeStatus map[NodeName]*Status

// Nodes is a map NodeName : Node
type Nodes map[NodeName]*Node

func (n *Node) CommandExist(command CommandName) bool {
	_, ok := n.Commands[command]
	return ok
}

func (n *Node) Status() *Status {
	status := NewStatusFailed("")
	for _, aMethod := range n.HTTPAccess {
		err := aMethod.do()
		if err != nil {
			n.Alive = false
			status.Error = err.Error()
			status.SetListStatus("HTTPAccess", StatusFailed)
			return status
		}
		status.SetListStatus("HTTPAccess", StatusPassed)
	}
	if n.Commands != nil {
		for name, command := range n.Commands {
			if command.GetStatus {
				out, err := n.run(command.Stdin)
				if err != nil {
					n.Alive = false
					status.Error = err.Error()
					status.SetListStatus(name, StatusFailed)
					return status
				}
				if command.Stdout != "" && command.Stdout != out {
					n.Alive = false
					status.SetListStatus(name, StatusFailed)
					return status
				} else if command.Stdout == "" {
					status.SetListStatus(name, StatusValue(out))
				} else {
					status.SetListStatus(name, StatusPassed)
				}
			}
		}
	}
	status.GeneralStatus = StatusRunning
	n.Alive = true
	return status
}

func (n *Node) Go(name NodeName, event chan Event) error {
	for {
		if n.Settings.Timeout <= 0 {
			event <- Event{name, n.Status()}
			break
		}
		status, d := n.Status(), time.Duration(n.Settings.Timeout)*time.Second
		status.NextUpdate = time.Now().Add(d).String()
		event <- Event{name, status}
		time.Sleep(d)
	}
	return nil
}

func (n *Node) run(command string) (string, error) {
	if n.Romote {
		client, err := ssh.New(n.Connection.User, n.Connection.Address, ssh.Key(n.Connection.SSHKey, n.Connection.PassPhrase))
		if err != nil {
			return "", err
		}
		out, err := client.Run(command)
		if err != nil {
			return "", err
		}
		err = client.Close()
		if err != nil {
			return "", err
		}
		return string(out), nil
	}
	out, err := exec.Command("bash", "-c", command).Output() // works for darwin
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (n *Node) valid() error {
	for _, httpAccess := range n.HTTPAccess {
		if err := httpAccess.valid(); err != nil {
			return err
		}
	}
	if n.OS != OSDarwin && n.OS != OSLinux && n.OS != OSWindows {
		return errors.New("Node validation: unknown OS")
	}
	if _, ok := SettingValueMap[n.Settings.Restart]; !ok {
		n.Settings.Restart = "never"
	}
	if _, ok := SettingValueMap[n.Settings.Reload]; !ok {
		n.Settings.Reload = "never"
	}
	if n.Settings.Timeout <= 0 {
		n.Settings.Timeout = -1
	}
	return nil
}

func (h *HTTPAccess) valid() error {
	_, ok := HttpMethodMap[h.Method]
	if !ok {
		return errors.New("HTTPAccess: unknown method")
	}
	_, err := url.ParseRequestURI(h.Address)
	if err != nil {
		return errors.New("HTTPAccess: can't parse url")
	}
	if h.StatusCode < 100 || h.StatusCode > 526 {
		return errors.New("HTTPAccess: unknown status code")
	}
	return nil
}

func (h *HTTPAccess) do() error {
	request, err := http.NewRequest(h.Method, h.Address, nil)
	if err != nil {
		return fmt.Errorf("HTTP access method: %s", err.Error())
	}
	if h.Headers != nil {
		for key, value := range h.Headers {
			request.Header.Set(key, value)
		}
	}
	client := new(http.Client)
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("HTTP access method: %s", err.Error())
	}
	if resp.StatusCode != h.StatusCode {
		return fmt.Errorf("HTTP access method: expected status code %d, got %d", h.StatusCode, resp.StatusCode)
	}
	return nil
}
