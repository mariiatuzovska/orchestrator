package orchestrator

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type Service struct {
	*ServiceConfiguration
	status StatusDetail
	node   map[string]*Node
}

type ServiceConfiguration struct {
	Name             string
	URL              string
	StartImmediately bool          // starts service immediately
	HTTPAccess       []*HTTPAccess // http access settings
	Nodes            []string      // array of node names
	Timeout          int           // seconds
}

// HTTPAccess smth like in consul config
type HTTPAccess struct {
	Method     string
	Address    string
	StatusCode int
	Headers    map[string]string
}

func NewService(config *ServiceConfiguration, nodes map[string]*Node) (*Service, error) {
	s := &Service{config, make(StatusDetail), make(map[string]*Node)}
	for _, nodName := range config.Nodes {
		if node, ok := nodes[nodName]; ok {
			s.node[nodName] = node
		}
	}
	return s, s.Valid()
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}

func (s *Service) Status() (*StatusDetail, error) {
	return &StatusDetail{}, nil
}

// func (n *Node) Go(name NodeName, event chan Event) error {
// 	for {
// 		if n.Settings.Timeout <= 0 {
// 			event <- Event{name, n.Status()}
// 			break
// 		}
// 		status, d := n.Status(), time.Duration(n.Settings.Timeout)*time.Second
// 		status.NextUpdate = time.Now().Add(d).String()
// 		event <- Event{name, status}
// 		time.Sleep(d)
// 	}
// 	return nil
// }

// func (n *Node) Run(command CommandName) (string, error) {
// 	if !n.CommandExist(command) {
// 		return "", fmt.Errorf("%s command is not exist in current node", command)
// 	}
// 	out, err := n.run(n.Commands[command].Stdin)
// 	if err != nil {
// 		return "", err
// 	}
// 	return out, nil
// }

// func (n *Node) run(command string) (string, error) {
// 	if n.Romote {
// 		client, err := ssh.New(n.Connection.User, n.Connection.Address, ssh.Key(n.Connection.SSHKey, n.Connection.PassPhrase))
// 		if err != nil {
// 			return "", err
// 		}
// 		out, err := client.Run(command)
// 		if err != nil {
// 			return "", err
// 		}
// 		err = client.Close()
// 		if err != nil {
// 			return "", err
// 		}
// 		return string(out), nil
// 	}
// 	out, err := exec.Command("bash", "-c", command).Output() // works for darwin
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(out), nil
// }

func (s *Service) Valid() error {
	for _, nodeName := range s.Nodes {
		node, ok := s.node[nodeName]
		if !ok {
			return fmt.Errorf("%s node is not defined", nodeName)
		}
		if err := node.Valid(); err != nil {
			return err
		}
	}
	for _, hAccess := range s.HTTPAccess {
		if err := hAccess.valid(); err != nil {
			return err
		}
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
