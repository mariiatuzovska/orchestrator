package orchestrator

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/ssh"
)

var (
	ServiceName string = "orchestrator"
	Version            = "0.0.5"
)

type Orchestrator struct {
	mux      sync.Mutex
	c        chan Event
	config   *Configuration
	nodes    map[string]*Node
	services map[string]*Service
	remote   map[string]*ssh.Client
}

type Event struct {
	Service string
	Status  ServiceStatusInfo
	// Error   error
}

func NewOrchestrator(config *Configuration) (*Orchestrator, error) {
	nodeMap, srvMap, remote := make(map[string]*Node), make(map[string]*Service), make(map[string]*ssh.Client)
	for _, nodConfig := range *config.Nodes {
		node, err := NewNode(nodConfig)
		if err != nil {
			return nil, err
		}
		if _, exist := nodeMap[node.NodeName]; exist {
			return nil, fmt.Errorf("Orchestrator: %s node is not unique / already defined / duplicated in NodeConfigurationArray", node.NodeName)
		}
		nodeMap[node.NodeName] = node
		if node.Connection != nil {
			client, err := node.Connect()
			if err == nil {
				remote[node.NodeName] = client
				nodeMap[node.NodeName].NodeStatus = StatusConnected
			} else {
				log.Printf("Orshestartor: %s node: %s\n", node.NodeName, err.Error())
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
			return nil, fmt.Errorf("Orchestrator: %s service is not unique / already defined / duplicated in ServiceConfigurationArray", srv.ServiceName)
		}
		srvMap[srv.ServiceName] = srv
	}
	return &Orchestrator{sync.Mutex{}, make(chan Event, 100), config, nodeMap, srvMap, remote}, nil
}

func (o *Orchestrator) StartOrchestrator(address string) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// SERVICES
	e.GET("/orchestrator/services", o.GetServicesController)
	e.POST("/orchestrator/services", o.CreateServiceController)
	e.PUT("/orchestrator/services", o.UpdateServiceController)
	e.GET("/orchestrator/services/:ServiceName", o.GetServiceByNameController)
	e.DELETE("/orchestrator/services/:ServiceName", o.DeleteServiceController)
	// SERVICES: START / STOP
	e.POST("/orchestrator/services/:ServiceName/:NodeName", o.StartServiceByNameController)
	e.DELETE("/orchestrator/services/:ServiceName/:NodeName", o.StopServiceByNameController)
	// NODES
	e.GET("/orchestrator/nodes", o.GetNodesController)
	e.POST("/orchestrator/nodes", o.CreateNodeController)
	e.PUT("/orchestrator/nodes", o.UpdateNodeController)
	e.GET("/orchestrator/nodes/:NodeName", o.GetNodeByNameController)
	e.DELETE("/orchestrator/nodes/:NodeName", o.DeleteNodeByNameController)
	// STATUSES
	e.GET("/orchestrator/statuses", o.GetServiceStatusesController)
	e.GET("/orchestrator/statuses/:ServiceName", o.GetServiceStatusByNameController)
	// ORCHESTRATOR
	go o.OrchestratorRoutine()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return e.Start(address)
}

func (o *Orchestrator) OrchestratorRoutine() {
	time.Sleep(time.Duration(5) * time.Second)
	for _, srv := range o.services {
		go o.ServiceStatusRoutine(srv)
	}
	for {
		e := <-o.c
		o.mux.Lock()
		_, ok := o.services[e.Service]
		if ok {
			o.services[e.Service].StatusInfo = e.Status
		}
		o.mux.Unlock()
	}
}

func (o *Orchestrator) ServiceStatusRoutine(srv *Service) {
	if srv.StartImmediately {
		for _, node := range srv.Nodes {
			err := o.StartService(node.NodeName, srv.ServiceName)
			if err != nil {
				log.Printf("Orchestrator: %s service has been started immediately on %s node. Error message: %s\n",
					srv.ServiceName, node.NodeName, err.Error())
			}
		}
	}
	for { // func ServiceStatus is mutual excluded
		status, err := o.ServiceStatus(srv.ServiceName) // returns error if only service is unknown
		if err != nil {                                 // in case on nil service -- routine stops
			break
		}
		o.c <- Event{srv.ServiceName, *status}
		if srv.Timeout <= 0 {
			break
		}
		time.Sleep(time.Duration(srv.Timeout) * time.Second)
	}
}

func (o *Orchestrator) ServiceStatus(service string) (*ServiceStatusInfo, error) {
	o.mux.Lock() // do not forget about mutual exclusion
	if _, ok := o.services[service]; !ok {
		o.mux.Unlock() // here
		return nil, fmt.Errorf("Service access: %s service is unknown", service)
	}
	info := &ServiceStatusInfo{StatusInactive, StatusUndefined, make([]*NodeStatusInfo, 0), time.Now().String(), ""}
	if o.services[service].Timeout > 0 {
		info.NextUpdate = time.Now().Add(time.Duration(o.services[service].Timeout) * time.Second).String()
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
		case OSLinux:
			command := fmt.Sprintf(LinuxTryIsActiveFormatString, service)
			out, err := o.runcommand(node.NodeName, command)
			if err != nil {
				nodStatus.ServiceStatus = StatusDisconnected
			} else if !strings.Contains(out, "0") {
				nodStatus.ServiceStatus = StatusInactive
			}
		case OSDarwin:
			command := fmt.Sprintf(DarwinTryIsActiveFormatString, service)
			out, err := o.runcommand(node.NodeName, command)
			if err != nil {
				nodStatus.ServiceStatus = StatusDisconnected
			} else if !strings.Contains(out, "0") {
				nodStatus.ServiceStatus = StatusInactive
			}
		default:
			nodStatus.ServiceStatus = StatusUnknownOS
		}
		info.NodeStatus = append(info.NodeStatus, nodStatus)
	}
	o.mux.Unlock() // here
	return info, nil
}

func (o *Orchestrator) StartService(node, service string) error {
	o.mux.Lock() // do not forget about mutual exclusion
	command := ""
	if _, ok := o.nodes[node]; !ok {
		o.mux.Unlock() // here
		return fmt.Errorf("Node access: %s node is unknown", node)
	}
	switch o.nodes[node].OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStartServiceFormatString, service)
	case OSLinux:
		command = fmt.Sprintf(LinuxStartServiceFormatString, service)
	}
	_, err := o.runcommand(node, command)
	o.mux.Unlock() // here
	return err
}

func (o *Orchestrator) StopService(node, service string) error {
	o.mux.Lock() // do not forget about mutual exclusion
	command := ""
	if _, ok := o.nodes[node]; !ok {
		o.mux.Unlock() // here
		return fmt.Errorf("Node access: %s node is unknown", node)
	}
	switch o.nodes[node].OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStopServiceFormatString, service)
	case OSLinux:
		command = fmt.Sprintf(LinuxStopServiceFormatString, service)
	}
	_, err := o.runcommand(node, command)
	o.mux.Unlock() // here
	return err
}

func (o *Orchestrator) runcommand(node, command string) (string, error) {
	var out []byte
	if _, ok := o.nodes[node]; !ok {
		return "", fmt.Errorf("Node access: %s node is unknown", node)
	}
	if o.nodes[node].Connection == nil { // LOCAL
		var err error
		switch o.nodes[node].OS {
		case OSDarwin:
			out, err = exec.Command("bash", "-c", command).Output()
			if err != nil {
				return "", err
			}
		case OSLinux:
			out, err = exec.Command("bash", "-c", command).Output()
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("Remote connection is not provided for this OS")
		}
	} else { // REMOTE
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
		default:
			return "", fmt.Errorf("Remote connection is not provided for this OS")
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
