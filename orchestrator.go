package orchestrator

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	OrchestratorServiceName string = "orchestrator"
	Version                        = "0.0.3"
)

type Orchestrator struct {
	this         *Service
	mux          sync.Mutex
	config       *Configuration
	nodes        map[string]*Node
	services     map[string]*Service
	statusdetail map[string]*StatusDetail
}

type Event struct {
	ServiceName string
	Status      StatusDetail
}

type JSONMessage struct {
	Message string
}

type ServiceStatusInfo struct {
	ServiceName  string
	StatusDetail *StatusDetail
}

func NewOrchestrator(config *Configuration) (*Orchestrator, error) {
	nodeMap, srvMap, this := make(map[string]*Node), make(map[string]*Service), new(Service)
	this = nil
	for _, nodConfig := range config.Nodes {
		node, err := NewNode(nodConfig)
		if err != nil {
			return nil, err
		}
		if _, exist := nodeMap[node.NodeName]; exist {
			return nil, fmt.Errorf("Orchestrator: node %s is not unique / already defined", node.NodeName)
		}
		nodeMap[node.NodeName] = node
	}
	for _, srvConfig := range config.Services {
		srv, err := NewService(srvConfig, nodeMap)
		if err != nil {
			return nil, err
		}
		if _, exist := nodeMap[srv.ServiceName]; exist {
			return nil, fmt.Errorf("Orchestrator: service %s is not unique / already defined", srv.ServiceName)
		}
		if srv.ServiceType == OrchestratorServiceType {
			if this != nil {
				return nil, fmt.Errorf("Orchestrator: orchestrators service type must be unique")
			}
			this = srv
		}
		srvMap[srv.ServiceName] = srv
	}
	if this == nil {
		return nil, fmt.Errorf("Orchestrator: main service is undefined")
	}
	return &Orchestrator{this, sync.Mutex{}, config, nodeMap, srvMap, make(map[string]*StatusDetail)}, nil
}

func (o *Orchestrator) Start() error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/orchestrator/services", func(c echo.Context) error {
		m := make([]*ServiceInfo, 0)
		for srvName := range o.statusdetail {
			srvInfo, err := o.GetServiceInfo(srvName)
			if err != nil {
				return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
			}
			m = append(m, srvInfo)
		}
		return c.JSON(http.StatusOK, m)
	})
	e.GET("/orchestrator/services/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srvInfo, err := o.GetServiceInfo(name[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
		}
		return c.JSON(http.StatusOK, srvInfo)
	})
	e.GET("/orchestrator/nodes", func(c echo.Context) error {
		nodes := make([]string, 0)
		for _, node := range o.config.Nodes {
			nodes = append(nodes, node.NodeName)
		}
		nodInfos, err := o.GetNodeInfos(nodes...)
		if err != nil {
			return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
		}
		return c.JSON(http.StatusOK, nodInfos)
	})
	e.GET("/orchestrator/nodes/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		nodInfos, err := o.GetNodeInfos(name[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
		}
		return c.JSON(http.StatusOK, nodInfos[0])
	})
	e.GET("/orchestrator/statuses", func(c echo.Context) error {
		statuses := make([]ServiceStatusInfo, 0)
		for srvName := range o.statusdetail {
			srvInfo, err := o.GetServiceInfo(srvName)
			if err != nil {
				return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
			}
			statuses = append(statuses, ServiceStatusInfo{srvInfo.ServiceName, srvInfo.StatusDetail})
		}
		return c.JSON(http.StatusOK, statuses)
	})
	e.GET("/orchestrator/statuses/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srvInfo, err := o.GetServiceInfo(name[0])
		if err != nil {
			return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
		}
		return c.JSON(http.StatusOK, ServiceStatusInfo{srvInfo.ServiceName, srvInfo.StatusDetail})
	})
	go o.Go()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return e.Start(o.this.URL)
}

func (o *Orchestrator) GetServiceInfo(name string) (*ServiceInfo, error) {
	srv, ok := o.services[name]
	if !ok {
		return nil, fmt.Errorf("%s service is unknown", name)
	}
	status, ok := o.statusdetail[name]
	if !ok {
		return nil, fmt.Errorf("%s service is unknown", name)
	}
	nodeInfo, err := o.GetNodeInfos(srv.Nodes...)
	if err != nil {
		return nil, err
	}
	return &ServiceInfo{
		ServiceName:      srv.ServiceName,
		StatusDetail:     status,
		ServiceType:      srv.ServiceType,
		URL:              srv.URL,
		StartImmediately: srv.StartImmediately,
		HTTPAccess:       srv.HTTPAccess,
		Timeout:          srv.Timeout,
		Nodes:            nodeInfo,
	}, nil
}

func (o *Orchestrator) GetNodeInfos(names ...string) ([]*NodeInfo, error) {
	info := make([]*NodeInfo, 0)
	for _, name := range names {
		node, ok := o.nodes[name]
		if !ok {
			return nil, fmt.Errorf("%s node is unknown", name)
		}
		info = append(info, &NodeInfo{
			Connected:         node.isConnected,
			NodeConfiguration: node.NodeConfiguration,
		})
	}
	return info, nil
}

func (o *Orchestrator) Go() {
	time.Sleep(time.Duration(3) * time.Second)
	c := make(chan Event, 100)
	for _, srv := range o.services {
		go func(srv *Service, c chan Event) {
			for {
				sd := srv.Status()
				c <- Event{srv.ServiceName, *sd}
				if srv.Timeout <= 0 {
					break
				}
				time.Sleep(time.Duration(srv.Timeout) * time.Second)
			}
		}(srv, c)
	}
	for {
		e := <-c
		o.mux.Lock()
		o.statusdetail[e.ServiceName] = &e.Status
		o.mux.Unlock()
	}
}

// func (o *Orchestrator) Stop() error {
// 	return nil
// }
