package orchestrator

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/ssh"
)

var (
	OrchestratorServiceName string = "orchestrator"
	Version                        = "0.0.4"
)

type Orchestrator struct {
	mux      sync.Mutex
	config   *Configuration
	nodes    map[string]*Node
	services map[string]*Service
	remote   map[string]*ssh.Client
}

type Configuration struct {
	ServicesPath, NodesPath string
	Services                *ServiceConfigurationArray
	Nodes                   *NodeConfigurationArray
}

type JSONMessage struct {
	Message string
}

func NewOrchestrator(config *Configuration) (*Orchestrator, error) {
	nodeMap, srvMap, remote := make(map[string]*Node), make(map[string]*Service), make(map[string]*ssh.Client)
	for _, nodConfig := range *config.Nodes {
		node, err := NewNode(nodConfig)
		if err != nil {
			return nil, err
		}
		if _, exist := nodeMap[node.NodeName]; exist {
			return nil, fmt.Errorf("Orchestrator: node %s is not unique / already defined / duplicated in NodeConfigurationArray", node.NodeName)
		}
		nodeMap[node.NodeName] = node
		if node.Connection != nil {
			client, err := node.Connect()
			if err == nil {
				remote[node.NodeName] = client
				nodeMap[node.NodeName].NodeStatus = StatusConnected
			} else {
				nodeMap[node.NodeName].NodeStatus = StatusDisconnected
			}
		} else {
			nodeMap[node.NodeName].NodeStatus = StatusConnected
		}
	}
	for _, srvConfig := range *config.Services {
		srv, err := NewService(srvConfig, nodeMap)
		if err != nil {
			return nil, err
		}
		if _, exist := srvMap[srv.ServiceName]; exist {
			return nil, fmt.Errorf("Orchestrator: service %s is not unique / already defined / duplicated in ServiceConfigurationArray", srv.ServiceName)
		}
		srvMap[srv.ServiceName] = srv
	}
	return &Orchestrator{sync.Mutex{}, config, nodeMap, srvMap, remote}, nil
}

func (o *Orchestrator) StartOrchestrator(address string) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/orchestrator/services", func(c echo.Context) error {
		m := make([]*Service, 0)
		for _, service := range o.services {
			m = append(m, service)
		}
		return c.JSON(http.StatusOK, m)
	})
	e.GET("/orchestrator/services/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srvInfo, ok := o.services[name[0]]
		if !ok {
			return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
		}
		return c.JSON(http.StatusOK, srvInfo)
	})
	e.GET("/orchestrator/nodes", func(c echo.Context) error {
		m := make([]*Node, 0)
		for _, node := range o.nodes {
			m = append(m, node)
		}
		return c.JSON(http.StatusOK, m)
	})
	e.GET("/orchestrator/nodes/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		nodInfo, ok := o.nodes[name[0]]
		if !ok {
			return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
		}
		return c.JSON(http.StatusOK, nodInfo)
	})
	e.GET("/orchestrator/statuses", func(c echo.Context) error {
		statuses := make([]struct {
			ServiceName string
			StatusInfo  ServiceStatusInfo
		}, 0)
		for _, service := range o.services {
			statuses = append(statuses, struct {
				ServiceName string
				StatusInfo  ServiceStatusInfo
			}{
				ServiceName: service.ServiceName,
				StatusInfo:  service.StatusInfo,
			})
		}
		return c.JSON(http.StatusOK, statuses)
	})
	e.GET("/orchestrator/statuses/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srvInfo, ok := o.services[name[0]]
		if !ok {
			return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
		}
		return c.JSON(http.StatusOK, struct {
			ServiceName string
			StatusInfo  ServiceStatusInfo
		}{
			ServiceName: srvInfo.ServiceName,
			StatusInfo:  srvInfo.StatusInfo,
		})
	})
	go o.Go()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return e.Start(address)
}

func (o *Orchestrator) ServiceStatus(service string) (*ServiceStatusInfo, error) {
	info := &ServiceStatusInfo{StatusUndefined, StatusUndefined, make([]*NodeStatusInfo, 0)}
	if _, ok := o.services[service]; !ok {
		return nil, fmt.Errorf("Service access: %s service is unknown", service)
	}
	if len(o.services[service].HTTPAccess) > 0 {
		info.HTTPAccessStatus = StatusPassed
		for _, access := range o.services[service].HTTPAccess {
			err := access.do()
			if err != nil {
				info.HTTPAccessStatus = StatusFailed
				continue
			}
		}
	}
	if info.HTTPAccessStatus == StatusPassed {
		info.ServiceStatus = StatusActive
	}
	for _, node := range o.services[service].Nodes {
		nodStatus := &NodeStatusInfo{node.NodeName, StatusActive}
		switch node.OS {
		case OSWindows:
			nodStatus.Status = StatusUnknownOS
		case OSLinux:
			command := fmt.Sprintf(LinuxTryIsActiveFormatString, service)
			out, err := o.runcommand(node.NodeName, command)
			if err != nil {
				nodStatus.Status = StatusDisconnected
			} else if !strings.Contains(out, "0") {
				nodStatus.Status = StatusInactive
			}
		case OSDarwin:
			command := fmt.Sprintf(DarwinTryIsActiveFormatString, service)
			out, err := o.runcommand(node.NodeName, command)
			if err != nil {
				nodStatus.Status = StatusDisconnected
			} else if !strings.Contains(out, "0") {
				nodStatus.Status = StatusInactive
			}
		}
		if nodStatus.Status == StatusActive {
			info.ServiceStatus = StatusActive
		}
		info.NodeStatus = append(info.NodeStatus, nodStatus)
	}
	return info, nil
}

func (o *Orchestrator) StartService(node, service string) error {
	command := ""
	if _, ok := o.nodes[node]; !ok {
		return fmt.Errorf("Node access: %s node is unknown", node)
	}
	switch o.nodes[node].OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStartServiceFormatString, service)
	case OSLinux:
		command = fmt.Sprintf(LinuxStartServiceFormatString, service)
	}
	_, err := o.runcommand(node, command)
	return err
}

func (o *Orchestrator) StopService(node, service string) error {
	command := ""
	if _, ok := o.nodes[node]; !ok {
		return fmt.Errorf("Node access: %s node is unknown", node)
	}
	switch o.nodes[node].OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStopServiceFormatString, service)
	case OSLinux:
		command = fmt.Sprintf(LinuxStopServiceFormatString, service)
	}
	_, err := o.runcommand(node, command)
	return err
}

func (o *Orchestrator) runcommand(node, command string) (string, error) {
	var out []byte
	if _, ok := o.nodes[node]; !ok {
		return "", fmt.Errorf("Node access: %s node is unknown", node)
	}
	if o.nodes[node].Connection == nil {
		var err error
		out, err = exec.Command("bash", "-c", command).Output()
		if err != nil {
			return "", err
		}
	} else {
		client, err := o.isConnected(node)
		if err != nil {
			return "", err
		}
		session, err := client.NewSession()
		if err != nil {
			return "", err
		}
		defer session.Close()
		switch o.nodes[node].OS {
		case OSDarwin:
			out, err = session.CombinedOutput(command)
			if err != nil {
				return "", err
			}
		case OSLinux:
			out, err = session.CombinedOutput(command)
			if err != nil {
				return "", err
			}
		case OSWindows:
			return "", fmt.Errorf("Remote connection is not provided for windows")
		}
	}
	return string(out), nil
}

func (o *Orchestrator) isConnected(node string) (*ssh.Client, error) {
	if _, ok := o.nodes[node]; !ok {
		return nil, fmt.Errorf("Node access: %s node is unknown", node)
	}
	client, ok := o.remote[node]
	if !ok {
		o.nodes[node].NodeStatus = StatusDisconnected
		return nil, fmt.Errorf("Node access: %s node has nil Connection", node)
	}
	session, err := client.NewSession()
	if err != nil {
		o.nodes[node].NodeStatus = StatusDisconnected
		return nil, err
	}
	err = session.Close()
	if err != nil {
		o.nodes[node].NodeStatus = StatusDisconnected
		return nil, err
	}
	return client, nil
}

func (o *Orchestrator) Go() {
	time.Sleep(time.Duration(5) * time.Second)
	type Event struct {
		Service string
		Status  ServiceStatusInfo
		Error   error
	}
	c := make(chan Event, 100)
	for _, srv := range o.services {
		for _, node := range srv.Nodes {
			go func(srv *Service, node *Node, c chan Event) {
				for {
					o.mux.Lock()
					status, err := o.ServiceStatus(srv.ServiceName)
					o.mux.Unlock()
					c <- Event{srv.ServiceName, *status, err}
					if srv.Timeout <= 0 {
						break
					}
					time.Sleep(time.Duration(srv.Timeout) * time.Second)
				}
			}(srv, node, c)
		}
	}
	for {
		e := <-c
		o.mux.Lock()
		o.services[e.Service].StatusInfo = e.Status
		if e.Error != nil {
			log.Println(e.Service, e.Status.ServiceStatus, e.Error.Error())
		}
		o.mux.Unlock()
	}
}
