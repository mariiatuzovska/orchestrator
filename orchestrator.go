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
	status     map[ServiceName]*ServiceStatus
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
	o := &Orchestrator{*config, make(map[ServiceName]*ServiceStatus), make(NodeStatus), make(chan Event, 100)}
	for srvName, srv := range *config {
		if _, exist := o.status[srvName]; exist {
			return nil, fmt.Errorf("Orchestrator: service %s is not unique / already defined", srvName)
		}
		for name, node := range srv.Nodes {
			if err := node.valid(); err != nil {
				return nil, fmt.Errorf("Orchestrator: %s: %s", name, err.Error())
			}
			if _, exist := o.nodestatus[name]; exist {
				return nil, fmt.Errorf("Orchestrator: node %s is not unique / already defined", name)
			}
			o.nodestatus[name] = NewInitializedStatus()
		}
		o.status[srvName] = &ServiceStatus{srv.DNS, &o.nodestatus}
	}
	return o, nil
}

func (o *Orchestrator) Start() error {
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
	}
}

func (o *Orchestrator) Stop() error {
	return nil
}
