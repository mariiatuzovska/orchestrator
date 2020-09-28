package orchestrator

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/ssh"
)

var (
	ServiceName string = "orchestrator"
	Version            = "0.1.1"
)

type Orchestrator struct {
	logLevel int
	mux      sync.Mutex
	ch       chan Event
	node     map[string]*Node
	service  map[string]*Service
	client   map[string]*ssh.Client // node's client
}

type Event struct {
	Service string
	Status  ServiceStatusInfo
	Error   error
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{1, sync.Mutex{}, make(chan Event, 100), make(map[string]*Node), make(map[string]*Service), make(map[string]*ssh.Client)}
}

func (o *Orchestrator) GetNode(name string) (*Node, error) {
	if node, exist := o.node[name]; exist {
		return node, nil
	}
	return nil, fmt.Errorf("Orchestrator: %s node is not exist", name)
}

func (o *Orchestrator) GetService(name string) (*Service, error) {
	if service, exist := o.service[name]; exist {
		return service, nil
	}
	return nil, fmt.Errorf("Orchestrator: %s service is not exist", name)
}

func (o *Orchestrator) SetNode(nodes ...*Node) error {
	tmp := o.node
	for _, node := range nodes {
		if err := node.Valid(); err != nil {
			return err
		}
		if _, exist := tmp[node.NodeName]; exist {
			return fmt.Errorf("Orchestrator: %s node already exist", node.NodeName)
		}
		tmp[node.NodeName] = node
		o.logf(Info, "New %s node has been set", node.NodeName)
	}
	o.node = tmp
	return nil
}

func (o *Orchestrator) SetService(services ...*Service) error {
	tmp := o.service
	for _, service := range services {
		if err := service.Valid(); err != nil {
			return err
		}
		for _, node := range service.Nodes {
			if _, exist := o.node[node.NodeName]; !exist {
				return fmt.Errorf("Orchestrator: %s node is not defined in orchestrator", node.NodeName)
			}
		}
		if _, exist := tmp[service.ServiceName]; exist {
			return fmt.Errorf("Orchestrator: %s service already exist", service.ServiceName)
		}
		tmp[service.ServiceName] = service
		o.logf(Info, "New %s service has been set", service.ServiceName)
	}
	o.service = tmp
	return nil
}

func (o *Orchestrator) DeleteNode(names ...string) error {
	for _, name := range names {
		if _, exist := o.node[name]; exist {
			for _, service := range o.service {
				for _, node := range service.Nodes {
					if node.NodeName == name {
						return fmt.Errorf("Orchestrator: %s node already in use", name)
					}
				}
			}
			delete(o.node, name)
			o.logf(Info, "%s node has been deleted from orchestrator", name)
		} else {
			return fmt.Errorf("Orchestrator: %s service is not exist", name)
		}
	}
	return nil
}

func (o *Orchestrator) DeleteService(names ...string) error {
	for _, name := range names {
		if _, exist := o.service[name]; exist {
			delete(o.service, name)
			o.logf(Info, "%s service has been deleted from orchestrator", name)
		} else {
			return fmt.Errorf("Orchestrator: %s service is not exist", name)
		}
	}
	return nil
}

func (o *Orchestrator) BasicAPI() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// SERVICES
	e.GET("/orchestrator/services", o.GetServicesController)
	e.GET("/orchestrator/services/:ServiceName", o.GetServiceByNameController)
	// SERVICES: START / STOP
	e.POST("/orchestrator/services/:ServiceName/:NodeName", o.StartServiceByNameController)
	e.DELETE("/orchestrator/services/:ServiceName/:NodeName", o.StopServiceByNameController)
	// NODES
	e.GET("/orchestrator/nodes", o.GetNodesController)
	e.GET("/orchestrator/nodes/:NodeName", o.GetNodeByNameController)
	e.POST("/orchestrator/nodes/:NodeName", o.ConnectToNodeByNameController)
	// STATUSES
	e.GET("/orchestrator/statuses", o.GetServiceStatusesController)
	e.GET("/orchestrator/statuses/:ServiceName", o.GetServiceStatusByNameController)

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))

	return e
}

func (o *Orchestrator) Start() {
	time.Sleep(time.Duration(1) * time.Second)
	for _, srv := range o.service {
		go o.serviceStatusRoutine(srv.ServiceName)
	}
	for {
		e := <-o.ch
		o.mux.Lock()
		_, ok := o.service[e.Service]
		if ok {
			o.service[e.Service].serviceStatus = e.Status
			if o.logLevel < Warning {
				o.logf(Info, "%s service has HTTP access status=%d", e.Service, e.Status.HTTPAccessStatus)
				for _, nodeStatus := range e.Status.NodeStatus {
					o.logf(Info, "%s service has status=%d on %s node", e.Service, nodeStatus.ServiceStatus, nodeStatus.NodeName)
				}
			} else {
				o.logf(Warning, "%s service has status=%d", e.Service, e.Status.ServiceStatus)
			}
		}
		o.mux.Unlock()
	}
}

func (o *Orchestrator) serviceStatusRoutine(serviceName string) error {
	srv, exist := o.service[serviceName]
	if !exist {
		fmt.Errorf("Service access: %s service is unknown", serviceName)
	}
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
			o.ch <- Event{srv.ServiceName, *status, err}
			return err
		}
		o.ch <- Event{srv.ServiceName, *status, nil}
		if srv.TimeoutSeconds < 1 {
			return nil
		}
		time.Sleep(time.Duration(srv.TimeoutSeconds) * time.Second)
	}
}

func (o *Orchestrator) ServiceStatus(serviceName string) (*ServiceStatusInfo, error) {
	o.mux.Lock()
	defer o.mux.Unlock()
	if _, ok := o.service[serviceName]; !ok {
		return nil, fmt.Errorf("Service access: %s service is unknown", serviceName)
	}
	info := &ServiceStatusInfo{StatusInactive, StatusUndefined, make([]*NodeStatusInfo, 0), time.Now().String(), ""}
	if o.service[serviceName].TimeoutSeconds > 0 {
		info.NextUpdate = time.Now().Add(time.Duration(o.service[serviceName].TimeoutSeconds) * time.Second).String()
	}
	if len(o.service[serviceName].HTTPAccess) > 0 {
		info.HTTPAccessStatus = StatusPassed
		for _, access := range o.service[serviceName].HTTPAccess {
			err := access.Do()
			if err != nil {
				info.HTTPAccessStatus = StatusFailed
				continue
			}
		}
	}
	if info.HTTPAccessStatus == StatusPassed {
		info.ServiceStatus = StatusActive
	}
	for _, node := range o.service[serviceName].Nodes {
		nodStatus := &NodeStatusInfo{node.NodeName, StatusInitialized}
		command := ""
		switch node.OS {
		case OSLinux:
			command = fmt.Sprintf(LinuxTryIsActiveFormatString, serviceName)
		case OSDarwin:
			command = fmt.Sprintf(DarwinTryIsActiveFormatString, serviceName)
		default:
			nodStatus.ServiceStatus = StatusUnknownOS
			continue
		}
		if command != "" {
			out, err := o.RunCommand(node.NodeName, command)
			if err != nil {
				nodStatus.ServiceStatus = StatusDisconnected
			} else if numStatus, err := strconv.Atoi(out); err == nil {
				nodStatus.ServiceStatus = numStatus
				if numStatus == StatusActive {
					info.ServiceStatus = StatusActive
				}
			} else {
				info.ServiceStatus = StatusInactive
			}
			info.NodeStatus = append(info.NodeStatus, nodStatus)
		}
	}
	return info, nil
}

func (o *Orchestrator) StartService(nodeName, serviceName string) error {
	o.mux.Lock()
	defer o.mux.Unlock()
	command := ""
	if _, ok := o.node[nodeName]; !ok {
		return fmt.Errorf("Node access: %s node is unknown", serviceName)
	}
	switch o.node[nodeName].OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStartServiceFormatString, serviceName)
	case OSLinux:
		command = fmt.Sprintf(LinuxStartServiceFormatString, serviceName)
	}
	_, err := o.RunCommand(serviceName, command)
	return err
}

func (o *Orchestrator) StopService(nodeName, serviceName string) error {
	o.mux.Lock()
	defer o.mux.Unlock()
	command := ""
	if _, ok := o.node[nodeName]; !ok {
		return fmt.Errorf("Node access: %s node is unknown", nodeName)
	}
	switch o.node[nodeName].OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStopServiceFormatString, serviceName)
	case OSLinux:
		command = fmt.Sprintf(LinuxStopServiceFormatString, serviceName)
	}
	_, err := o.RunCommand(nodeName, command)
	return err
}

func (o *Orchestrator) RunCommand(nodeName, command string) (string, error) {
	var out []byte
	if _, ok := o.node[nodeName]; !ok {
		return "", fmt.Errorf("Node access: %s node is unknown", nodeName)
	}
	if o.node[nodeName].Connection == nil { // LOCAL
		var err error
		switch o.node[nodeName].OS {
		case OSDarwin, OSLinux:
			out, err = exec.Command("bash", "-c", command).Output()
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("Remote connection is not provided for %s OS", o.node[nodeName].OS)
		}
	} else { // REMOTE
		client, err := o.IsNodeConnected(nodeName)
		if err != nil {
			return "", err
		}
		session, err := client.NewSession()
		if err != nil {
			return "", err
		}
		defer session.Close()
		switch o.node[nodeName].OS {
		case OSDarwin, OSLinux:
			out, err = session.CombinedOutput(command)
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("Remote connection is not provided for %s OS", o.node[nodeName].OS)
		}
	}
	return string(out), nil
}

func (o *Orchestrator) IsNodeConnected(nodeNmae string) (*ssh.Client, error) {
	if _, ok := o.node[nodeNmae]; !ok {
		return nil, fmt.Errorf("Node access: %s node is unknown", nodeNmae)
	}
	client, ok := o.client[nodeNmae]
	if !ok {
		o.node[nodeNmae].nodeStatus = StatusDisconnected
		return nil, fmt.Errorf("Node access: %s node has nil Connection", nodeNmae)
	}
	session, err := client.NewSession()
	if err != nil {
		o.node[nodeNmae].nodeStatus = StatusDisconnected
		return nil, err
	}
	err = session.Close()
	if err != nil {
		o.node[nodeNmae].nodeStatus = StatusDisconnected
		return nil, err
	}
	return client, nil
}

func (o *Orchestrator) SetLogLevel(lvl int) {
	if lvl > -1 && lvl < 5 {
		o.logLevel = lvl
	}
}

func (o *Orchestrator) logf(lvl int, format string, msg ...interface{}) {
	o.mux.Lock()
	var logLevels = map[int]string{Debug: "Debug", Info: "Info", Warning: "Warning", Error: "Error", Fatal: "Fatal"}
	if t, ok := logLevels[lvl]; ok && lvl >= o.logLevel {
		log.Printf("%s | Orchestrator | %s | %s", time.Now().Format(time.RFC3339), t, fmt.Sprintf(format, msg...))
	}
	o.mux.Unlock()
}
