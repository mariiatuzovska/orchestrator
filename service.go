package orchestrator

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

type Service struct {
	serviceStatus ServiceStatusInfo
	ServiceInfo
	Nodes []*Node
}

type ServiceInfo struct {
	ServiceName      string
	ServiceType      string
	URL              string
	StartImmediately bool          // starts service immediately
	HTTPAccess       []*HTTPAccess // http access settings
	TimeoutSeconds   int           // seconds
}

type ServiceConfiguration struct {
	ServiceInfo
	NodeNames []string // array of node names
}

type ServiceStatusInfo struct {
	ServiceStatus    int
	HTTPAccessStatus int
	NodeStatus       []*NodeStatusInfo
	ThisUpdate       string
	NextUpdate       string
}

type NodeStatusInfo struct {
	NodeName      string
	ServiceStatus int
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

func NewService(config *ServiceConfiguration) *Service {
	return &Service{ServiceStatusInfo{StatusInitialized, StatusInitialized, []*NodeStatusInfo{}, "", ""}, config.ServiceInfo, []*Node{}}
}

func (s *Service) Status() *ServiceStatusInfo {
	status := s.serviceStatus
	return &status
}

func (s *Service) SetNode(node *Node) error {
	if s.Nodes == nil || len(s.Nodes) < 1 {
		var nodes []*Node
		s.Nodes = nodes
	}
	for _, n := range s.Nodes {
		if n.NodeName == node.NodeName {
			return fmt.Errorf("Service validation: %s node already exist", node.NodeName)
		}
	}
	if err := node.Valid(); err != nil {
		return err
	}
	s.Nodes = append(s.Nodes, node)
	return nil
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
	if s.TimeoutSeconds < 1 {
		s.TimeoutSeconds = 0
	}
	for _, node := range s.Nodes {
		if err := node.Valid(); err != nil {
			return fmt.Errorf("Service validation: %s node is not valid: %s", node.NodeName, err.Error())
		}
	}
	for _, hAccess := range s.HTTPAccess {
		if err := hAccess.Valid(); err != nil {
			return err
		}
	}
	return nil
}

func (h *HTTPAccess) Valid() error {
	_, ok := HttpMethodMap[h.Method]
	if !ok {
		return errors.New("HTTPAccess validation: unknown method")
	}
	_, err := url.ParseRequestURI(h.Address)
	if err != nil {
		return errors.New("HTTPAccess validation: can't parse url")
	}
	if h.StatusCode < 100 || h.StatusCode > 526 {
		return errors.New("HTTPAccess validation: unknown status code")
	}
	return nil
}

func (h *HTTPAccess) Do() error {
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
