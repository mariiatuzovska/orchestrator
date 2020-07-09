package orchestrator

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	OrchestratorServiceName ServiceName = "orchestrator"
	Version                             = "0.0.1"
)

type Orchestrator struct {
	service    Configuration
	status     map[ServiceName]ServiceStatus
	nodestatus NodeStatus
	event      chan Event
}

type Event struct {
	Name   NodeName
	Status *Status
}

func NewOrchestrator(config *Configuration) (*Orchestrator, error) {
	if !config.Valid() {
		return nil, errors.New("Orchestrator: configuration file is not valid")
	}
	o := &Orchestrator{*config, make(map[ServiceName]ServiceStatus), make(NodeStatus), make(chan Event, 100)}
	for srvName, srv := range *config {
		if _, exist := o.status[srvName]; exist {
			return nil, fmt.Errorf("Orchestrator: service %s is not unique / already defined", srvName)
		}
		o.status[srvName] = ServiceStatus{srv.URL, make(map[NodeName]StatusValue)}
		for name, node := range srv.Nodes {
			if err := node.valid(); err != nil {
				return nil, fmt.Errorf("Orchestrator: %s: %s", name, err.Error())
			}
			if _, exist := o.nodestatus[name]; exist {
				return nil, fmt.Errorf("Orchestrator: node %s is not unique / already defined", name)
			}
			srv.Nodes[name].Alive = false
			o.nodestatus[name] = NewStatusInitialized()
			o.status[srvName].NodeStatuses[name] = o.nodestatus[name].GeneralStatus
		}
	}
	o.UpdateServiceStatus()
	return o, nil
}

func (o *Orchestrator) UpdateServiceStatus() {
	for srvName, srv := range o.service {
		for name := range srv.Nodes {
			o.status[srvName].NodeStatuses[name] = o.nodestatus[name].GeneralStatus
		}
	}
}

func (o *Orchestrator) Start() error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/orchestrator/configuration", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.service)
	})
	e.GET("/orchestrator/configuration/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srv, ok := o.service[ServiceName(name[0])]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.JSON(http.StatusOK, srv)
	})
	e.GET("/orchestrator/services", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.status)
	})
	e.GET("/orchestrator/services/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srv, ok := o.status[ServiceName(name[0])]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.JSON(http.StatusOK, srv)
	})
	e.GET("/orchestrator/nodes", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.nodestatus)
	})
	e.GET("/orchestrator/nodes/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		nod, ok := o.nodestatus[NodeName(name[0])]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.JSON(http.StatusOK, nod)
	})
	go o.Go()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return e.Start(o.service[OrchestratorServiceName].URL)
}

func (o *Orchestrator) Go() error {
	for _, srv := range o.service {
		for name, node := range srv.Nodes {
			go node.Go(name, o.event)
		}
	}
	for {
		e := <-o.event
		_, ok := o.nodestatus[e.Name]
		if !ok {
			return errors.New("Orchestrator: undefined node")
		}
		o.nodestatus[e.Name] = e.Status
		o.UpdateServiceStatus()
	}
}

func (o *Orchestrator) Stop() error {
	return nil
}
