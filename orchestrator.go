package orchestrator

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	ServiceName string = "orchestrator"
	fMux        sync.Mutex
)

type Orchestrator struct {
	logLevel int
	ch       chan Event
	node     map[string]*Node
	service  map[string]*Service
	client   map[string]*ssh.Client // node's client
	status   map[string]bool
}

type Event struct {
	Service string
	Status  ServiceStatusInfo
	Error   error
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{ERROR, make(chan Event, 100), make(map[string]*Node), make(map[string]*Service), make(map[string]*ssh.Client), make(map[string]bool)}
}

func (o *Orchestrator) GetNode(name string) (*Node, error) {
	if node, exist := o.node[name]; exist {
		n := *node // copy
		return &n, nil
	}
	return nil, o.Errorf("'%s' node is not exist", name)
}

func (o *Orchestrator) GetService(name string) (*Service, error) {
	if service, exist := o.service[name]; exist {
		s := *service // copy
		return &s, nil
	}
	return nil, o.Errorf("'%s' service is not exist", name)
}

func (o *Orchestrator) copyNodesAsArray() []*Node {
	fMux.Lock()
	nodes := []*Node{}
	for _, node := range o.node {
		if node != nil {
			cp := *node
			nodes = append(nodes, &cp)
		}
	}
	fMux.Unlock()
	return nodes
}

func (o *Orchestrator) copyNodesAsMap() map[string]*Node {
	fMux.Lock()
	nodes := map[string]*Node{}
	for _, node := range o.node {
		if node != nil {
			cp := *node
			nodes[cp.NodeName] = &cp
		}
	}
	fMux.Unlock()
	return nodes
}

func (o *Orchestrator) copyServicesAsArray() []*Service {
	fMux.Lock()
	services := []*Service{}
	for _, srv := range o.service {
		if srv != nil {
			cp := *srv
			services = append(services, &cp)
		}
	}
	fMux.Unlock()
	return services
}

func (o *Orchestrator) copyServicesAsMap() map[string]*Service {
	fMux.Lock()
	services := map[string]*Service{}
	for _, srv := range o.service {
		if srv != nil {
			cp := *srv
			services[cp.ServiceName] = &cp
		}
	}
	fMux.Unlock()
	return services
}

func (o *Orchestrator) RegistrateNodes(nodes ...*Node) error {
	for _, node := range nodes {
		if err := node.Valid(); err != nil {
			return err
		}
		if _, exist := o.node[node.NodeName]; exist {
			return o.Errorf("'%s' node already exist", node.NodeName)
		}
		fMux.Lock()
		for _, n := range o.node {
			if n.Connection == nil && node.Connection == nil {
				fMux.Unlock()
				return o.Errorf("'%s' local node already exist", n.NodeName)
			} else if n.Connection != nil && node.Connection != nil {
				if n.Connection.Host == node.Connection.Host {
					fMux.Unlock()
					return o.Errorf("'%s' node already exist with same '%s' host", node.NodeName, node.Connection.Host)
				}
			}
		}
		o.node[node.NodeName] = node
		fMux.Unlock()
	}
	return nil
}

func (o *Orchestrator) RegistrateServices(services ...*Service) error {
	fMux.Lock()
	for _, service := range services {
		if _, exist := o.service[service.ServiceName]; exist {
			fMux.Unlock()
			return o.Errorf("'%s' service already exist", service.ServiceName)
		}
		if err := service.Valid(); err != nil {
			fMux.Unlock()
			return err
		}
		nodes := make([]*Node, len(service.Nodes))
		for i, node := range service.Nodes {
			if _, exist := o.node[node.NodeName]; !exist {
				fMux.Unlock()
				return o.Errorf("'%s' node is not defined in orchestrator", node.NodeName)
			}
			nodes[i] = o.node[node.NodeName]
		}
		o.service[service.ServiceName] = service
		o.service[service.ServiceName].Nodes = nodes
	}
	fMux.Unlock()
	return nil
}

func (o *Orchestrator) RemoveNodes(names ...string) error {
	for _, name := range names {
		if _, exist := o.node[name]; exist {
			for _, service := range o.service {
				for _, node := range service.Nodes {
					if node.NodeName == name {
						return o.Errorf("%s node already in use", name)
					}
				}
			}
			fMux.Lock()
			delete(o.node, name)
			fMux.Unlock()
			o.logf(INFO, "'%s' node has been deleted from orchestrator", name)
		} else {
			return o.Errorf("'%s' service is not exist", name)
		}
	}
	return nil
}

func (o *Orchestrator) RemoveService(names ...string) error {
	for _, name := range names {
		if _, exist := o.service[name]; exist {
			fMux.Lock()
			delete(o.service, name)
			fMux.Unlock()
			o.logf(INFO, "'%s' service has been deleted from orchestrator", name)
		} else {
			return o.Errorf("'%s' service is not exist", name)
		}
	}
	return nil
}

func (o *Orchestrator) Start() {
	time.Sleep(time.Duration(1) * time.Second)
	for _, srv := range o.service {
		go o.ServiceStatusRoutine(srv.ServiceName)
	}
	for {
		e := <-o.ch
		if _, ok := o.service[e.Service]; ok {
			fMux.Lock()
			o.service[e.Service].ServiceStatus = e.Status
			fMux.Unlock()
			if o.logLevel < INFO {
				o.logf(DEBUG, "'%s' service has HTTP access status=%d", e.Service, e.Status.HTTPAccessStatus)
				for _, nodeStatus := range e.Status.NodeStatus {
					o.logf(DEBUG, "'%s' service has status=%d on '%s' node", e.Service, nodeStatus.ServiceStatus, nodeStatus.NodeName)
				}
			}
			o.logf(INFO, "'%s' service has status=%d", e.Service, e.Status.ServiceStatus)
		}
	}
}

func (o *Orchestrator) ServiceStatusRoutine(serviceName string) {
	srv, err := o.GetService(serviceName)
	if err != nil {
		o.logf(ERROR, err.Error())
		return
	}
	fMux.Lock()
	if _, exist := o.status[serviceName]; exist {
		fMux.Unlock()
		o.logf(ERROR, "'%s' service already in use", serviceName)
		return
	}
	o.status[serviceName] = true
	fMux.Unlock()
	for { // func ServiceStatus is mutual excluded
		status, err := o.ServiceStatus(srv.ServiceName) // returns error if only service is unknown
		if err != nil {                                 // in case on nil service -- routine stops
			o.ch <- Event{srv.ServiceName, *status, err}
			fMux.Lock()
			delete(o.status, serviceName)
			fMux.Unlock()
			return
		}
		o.ch <- Event{srv.ServiceName, *status, nil}
		o.logf(DEBUG, "'%s' service has status=%d", srv.ServiceName, status.ServiceStatus)
		if srv.TimeoutSeconds < 1 {
			fMux.Lock()
			delete(o.status, serviceName)
			fMux.Unlock()
			return
		}
		time.Sleep(time.Duration(srv.TimeoutSeconds) * time.Second)
	}
}

func (o *Orchestrator) ServiceStatus(serviceName string) (*ServiceStatusInfo, error) {
	service, err := o.GetService(serviceName)
	if err != nil {
		return nil, err
	}
	info := &ServiceStatusInfo{StatusUndefined, StatusUndefined, make([]*NodeStatusInfo, 0), time.Now(), time.Time{}}
	if service.TimeoutSeconds > 0 {
		info.NextUpdate = time.Now().Add(time.Duration(service.TimeoutSeconds) * time.Second)
	}
	if len(service.HTTPAccess) > 0 {
		info.HTTPAccessStatus = StatusPassed
		for _, access := range service.HTTPAccess {
			err := access.Do()
			if err != nil {
				info.HTTPAccessStatus = StatusFailed
				continue
			}
		}
	}
	for _, node := range service.Nodes {
		n, err := o.GetNode(node.NodeName)
		if err != nil {
			return nil, err
		}
		nodStatus := &NodeStatusInfo{n.NodeName, node.NodeStatus, StatusUndefined}
		command := ""
		switch n.OS {
		case OSLinux:
			command = fmt.Sprintf(LinuxTryIsActiveFormatString, serviceName)
		case OSDarwin:
			command = fmt.Sprintf(DarwinTryIsActiveFormatString, serviceName)
		default:
			nodStatus.ServiceStatus = StatusUnknownOS
		}
		if command != "" {
			out, err := o.RunCommand(n.NodeName, command)
			if err != nil {
				o.logf(DEBUG, "Running command error: %s", err.Error())
				nodStatus.ServiceStatus = StatusDisconnected
				if info.ServiceStatus == StatusUndefined {
					info.ServiceStatus = StatusInactive
				}
			} else {
				outStr := strings.ReplaceAll(string(out), "\n", "")
				o.logf(DEBUG, "Status result by '%s' node: %s", n.NodeName, outStr)
				numStatus, err := strconv.Atoi(outStr)
				if err == nil {
					nodStatus.ServiceStatus = numStatus
				}
				if info.ServiceStatus == StatusUndefined && nodStatus.ServiceStatus == StatusActive {
					info.ServiceStatus = StatusActive
				}
			}
		}
		info.NodeStatus = append(info.NodeStatus, nodStatus)
	}
	return info, nil
}

func (o *Orchestrator) StartService(nodeName, serviceName string) error {
	command := ""
	service, err := o.GetService(serviceName)
	if err != nil {
		return err
	}
	node := new(Node)
	for _, n := range service.Nodes {
		if n.NodeName == nodeName {
			node = n
		}
	}
	switch node.OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStartServiceFormatString, serviceName)
	case OSLinux:
		command = fmt.Sprintf(LinuxStartServiceFormatString, serviceName)
	default:
		return o.Errorf("unknown node '%s' or node's OS '%s'", nodeName, node.OS)
	}
	if _, err := o.RunCommand(node.NodeName, command); err != nil {
		o.logf(ERROR, "'%s' service has not been started on '%s' node. Error message: %s", serviceName, nodeName, err.Error())
		return err
	}
	o.logf(WARNING, "'%s' service has been started on '%s' node", serviceName, nodeName)
	return nil
}

func (o *Orchestrator) StopService(nodeName, serviceName string) error {
	command := ""
	service, err := o.GetService(serviceName)
	if err != nil {
		return err
	}
	node := new(Node)
	for _, n := range service.Nodes {
		if n.NodeName == nodeName {
			node = n
		}
	}
	switch node.OS {
	case OSDarwin:
		command = fmt.Sprintf(DarwinStopServiceFormatString, serviceName)
	case OSLinux:
		command = fmt.Sprintf(LinuxStopServiceFormatString, serviceName)
	default:
		return o.Errorf("unknown '%s' node or node's OS '%s'", nodeName, node.OS)
	}
	if _, err := o.RunCommand(node.NodeName, command); err != nil {
		o.logf(ERROR, "'%s' service has not been started on '%s' node. Error message: %s", serviceName, nodeName, err.Error())
		return err
	}
	o.logf(WARNING, "'%s' service has been stopped on '%s' node", serviceName, nodeName)
	return nil
}

func (o *Orchestrator) ConnectNode(nodeName, passPhrase string) error {
	fMux.Lock()
	if _, ok := o.node[nodeName]; !ok {
		fMux.Unlock()
		return o.Errorf("unknown '%s' node", nodeName)
	}
	if o.node[nodeName].Connection != nil {
		client, err := o.node[nodeName].Connect(passPhrase)
		if err != nil {
			o.node[nodeName].NodeStatus = StatusDisconnected
			fMux.Unlock()
			return o.Errorf("'%s' node connection error: %s", nodeName, err.Error())
		}
		o.client[nodeName] = client
	}
	o.node[nodeName].NodeStatus = StatusConnected
	fMux.Unlock()
	o.logf(WARNING, "'%s' node has been connected", nodeName)
	return nil
}

func (o *Orchestrator) DisconnectNode(nodeName string) error {
	fMux.Lock()
	if _, ok := o.node[nodeName]; !ok {
		fMux.Unlock()
		return o.Errorf("unknown '%s' node", nodeName)
	}
	if o.node[nodeName].Connection != nil {
		delete(o.client, nodeName)
		o.node[nodeName].NodeStatus = StatusDisconnected
	} else {
		o.node[nodeName].NodeStatus = StatusConnected
	}
	fMux.Unlock()
	o.logf(WARNING, "'%s' node has been disconnected", nodeName)
	return nil
}

func (o *Orchestrator) RunCommand(nodeName, command string) ([]byte, error) {
	var out []byte
	node, err := o.GetNode(nodeName)
	if err != nil {
		return nil, err
	}
	if node.Connection == nil { // LOCAL
		var err error
		switch node.OS {
		case OSDarwin, OSLinux:
			o.logf(DEBUG, "'%s' node command: '%s'", nodeName, command)
			out, err = exec.Command("bash", "-c", command).Output()
			if err != nil {
				return nil, err
			}
		default:
			return nil, o.Errorf("remote connection is not provided for '%s' OS", o.node[nodeName].OS)
		}
	} else { // REMOTE
		client, err := o.IsNodeConnected(nodeName)
		if err != nil {
			return nil, err
		}
		session, err := client.NewSession()
		if err != nil {
			return nil, err
		}
		defer session.Close()
		switch node.OS {
		case OSDarwin, OSLinux:
			o.logf(DEBUG, "'%s' node command: '%s'", nodeName, command)
			out, err = session.CombinedOutput(command)
			if err != nil {
				return nil, err
			}
		default:
			return nil, o.Errorf("remote connection is not provided for %s OS", o.node[nodeName].OS)
		}
	}
	return out, nil
}

func (o *Orchestrator) IsNodeConnected(nodeName string) (*ssh.Client, error) {
	fMux.Lock()
	if _, ok := o.node[nodeName]; !ok {
		fMux.Unlock()
		return nil, o.Errorf("Node access: unknown '%s' node", nodeName)
	}
	client, ok := o.client[nodeName]
	if !ok {
		o.node[nodeName].NodeStatus = StatusDisconnected
		fMux.Unlock()
		return nil, o.Errorf("Node access: '%s' node has nil Connection", nodeName)
	}
	session, err := client.NewSession()
	if err != nil {
		o.node[nodeName].NodeStatus = StatusDisconnected
		fMux.Unlock()
		return nil, err
	}
	err = session.Close()
	if err != nil {
		o.node[nodeName].NodeStatus = StatusDisconnected
		fMux.Unlock()
		return nil, err
	}
	o.node[nodeName].NodeStatus = StatusConnected
	fMux.Unlock()
	return client, nil
}

func (o *Orchestrator) SetLogLevel(lvl int) {
	if lvl > -1 && lvl < 5 {
		o.logLevel = lvl
	} else {
		o.logLevel = INFO
	}
}

func (o *Orchestrator) logf(lvl int, format string, msg ...interface{}) {
	fMux.Lock()
	var logLevels = map[int]string{DEBUG: "Debug", INFO: "Info", WARNING: "Warning", ERROR: "Error"}
	if t, ok := logLevels[lvl]; ok && lvl >= o.logLevel {
		log.Printf("%s | Orchestrator | %s | %s", time.Now().Format(time.RFC3339), t, fmt.Sprintf(format, msg...))
	}
	fMux.Unlock()
}

func (o *Orchestrator) Errorf(format string, msg ...interface{}) error {
	return fmt.Errorf("Orchestrator: %s", fmt.Sprintf(format, msg...))
}
