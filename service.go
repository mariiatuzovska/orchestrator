package orchestrator

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type Service struct {
	*ServiceConfiguration
	status map[string]*StatusDetail
	node   map[string]*Node
}

type ServiceInfo struct {
	ServiceName      string
	ServiceType      string
	StatusDetail     *StatusDetail
	URL              string
	StartImmediately bool          // starts service immediately
	HTTPAccess       []*HTTPAccess // http access settings
	Timeout          int           // seconds
	Nodes            []*NodeInfo
}

type ServiceConfiguration struct {
	ServiceName      string
	ServiceType      string
	URL              string
	StartImmediately bool          // starts service immediately
	HTTPAccess       []*HTTPAccess // http access settings
	Timeout          int           // seconds
	Nodes            []string      // array of node names
}

// HTTPAccess smth like in consul config
type HTTPAccess struct {
	Method     string
	Address    string
	StatusCode int
	Headers    map[string]string
}

type StatusDetail struct {
	Active     bool
	HTTPAccess string
	Statuses   []*StatusInfo
}

type StatusInfo struct {
	NodeName      string
	ServiceStatus string
}

func NewService(config *ServiceConfiguration, nodes map[string]*Node) (*Service, error) {
	s := &Service{config, make(map[string]*StatusDetail), make(map[string]*Node)}
	for _, nodName := range config.Nodes {
		if node, ok := nodes[nodName]; ok {
			s.node[nodName] = node
		}
	}
	return s, s.Valid()
}

// func (s *Service) Start() error {
// 	return nil
// }

// func (s *Service) Stop() error {
// 	return nil
// }

func (s *Service) Status() *StatusDetail {
	sd := &StatusDetail{false, HTTPAccessStatusUndefined, make([]*StatusInfo, 0)}
	for _, access := range s.HTTPAccess {
		err := access.do()
		if err != nil {
			sd.HTTPAccess = err.Error()
		} else {
			sd.Active = true
			sd.HTTPAccess = HTTPAccessStatusPassed
		}
	}
	for nodName, node := range s.node {
		err := node.ServiceStatus(s.ServiceName)
		if err != nil {
			sd.Statuses = append(sd.Statuses, &StatusInfo{nodName, err.Error()})
		} else {
			sd.Active = true
			sd.Statuses = append(sd.Statuses, &StatusInfo{nodName, StatusActive})
		}
	}
	return sd
}

func (s *Service) Valid() error {
	if s.ServiceName == "" {
		return fmt.Errorf("Service validation: undefined service name")
	}
	if s.ServiceType == "" {
		return fmt.Errorf("Service validation: %s service has undefined service type", s.ServiceName)
	}
	if len(s.Nodes) < 1 {
		return fmt.Errorf("Service validation: %s service must include node(s)", s.ServiceName)
	}
	for _, nodeName := range s.Nodes {
		node, ok := s.node[nodeName]
		if !ok {
			return fmt.Errorf("Service validation: %s node is not defined for %s service", nodeName, s.ServiceName)
		}
		if err := node.Valid(); err != nil {
			return fmt.Errorf("Service validation: %s node is not valid: %s", nodeName, err.Error())
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
