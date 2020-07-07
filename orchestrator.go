package orchestartor

import (
	"errors"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	OrchestratorServiceName ServiceName = "orchestrator"
	Version                             = "0.0.1"
)

type Orchestartor struct {
	service    Configuration
	status     map[ServiceName]*ServiceStatus
	nodestatus map[NodeName]*NodeStatus
	event      chan NodeStatus
}

// Status - general structure for Services, Nodes and orchestrator
type Status struct {
	OK     bool // is running
	Status int
	Error  error // last fatal error or nil
}

type ServiceStatus struct {
	DNS   string
	Nodes []*NodeStatus
}

type NodeStatus struct {
	NodeName NodeName
	*Node
	Status                 *Status
	ThisUpdate, NextUpdate string
}

func NewOrchestrator(config *Configuration) (*Orchestartor, error) {
	if !config.Valid() {
		return nil, errors.New("Orchestartor: configuration file is not valid")
	}
	status, nodestatus := make(map[ServiceName]*ServiceStatus), make(map[NodeName]*NodeStatus)
	for srvName, srv := range *config {
		if _, exist := status[srvName]; exist {
			return nil, errors.New("Orchestartor: service name is not unique / already defined")
		}
		aNodeStatus := make([]*NodeStatus, 0)
		for _, node := range srv.Nodes {
			if !node.valid() {
				return nil, errors.New("Orchestartor: node is not valid")
			}
			name := NewNodeName(srvName, node.Key)
			if _, exist := nodestatus[name]; exist {
				return nil, errors.New("Orchestartor: node name is not unique / already defined")
			}
			nodestatus[name] = &NodeStatus{name, node, nil, "", ""}
			aNodeStatus = append(aNodeStatus, nodestatus[name])
		}
		status[srvName] = &ServiceStatus{srv.DNS, aNodeStatus}
	}
	return &Orchestartor{*config, status, nodestatus, make(chan NodeStatus, 100)}, nil
}

// func (o *Orchestartor) Valid() bool {
// 	return o.service[OrchestratorServiceName].Valid()
// }

// func (o *Orchestartor) Status() *Status {
// 	return o.service[OrchestratorServiceName].Status()
// }

func (o *Orchestartor) Start() error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/orchestrator/status", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.status)
	})
	e.GET("/orchestrator/service", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.service)
	})
	go o.Go()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return e.Start(o.service[OrchestratorServiceName].DNS)
}

func (o *Orchestartor) Go() error {
	for srvName, srv := range o.service {
		for _, node := range srv.Nodes {
			go node.Go(srvName, o.event)
		}
	}
	for {
		e := <-o.event
		nodestatus, ok := o.nodestatus[e.NodeName]
		if !ok {
			return errors.New("Orchestartor: undefined node")
		}
		nodestatus.Status = e.Status
	}
}

func (o *Orchestartor) Stop() error {
	return nil
}
