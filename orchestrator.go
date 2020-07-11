package orchestrator

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	OrchestratorServiceName string = "orchestrator"
	Version                        = "0.0.2"
)

type Orchestrator struct {
	config       *Configuration
	nodes        map[string]*Node
	services     map[string]*Service
	statusdetail map[string]*StatusDetail
}

type Event struct {
	ServiceName string
	Status      StatusDetail
}

type StatusDetail map[string]string

func NewOrchestrator(config *Configuration) (*Orchestrator, error) {
	nodeMap, srvMap := make(map[string]*Node), make(map[string]*Service)
	for _, nodConfig := range config.Nodes {
		node, err := NewNode(nodConfig)
		if err != nil {
			return nil, err
		}
		if _, exist := nodeMap[node.Name]; exist {
			return nil, fmt.Errorf("Orchestrator: node %s is not unique / already defined", node.Name)
		}
		nodeMap[node.Name] = node
	}
	for _, srvConfig := range config.Services {
		srv, err := NewService(srvConfig, nodeMap)
		if err != nil {
			return nil, err
		}
		if _, exist := nodeMap[srv.Name]; exist {
			return nil, fmt.Errorf("Orchestrator: service %s is not unique / already defined", srv.Name)
		}
		srvMap[srv.Name] = srv
	}
	return &Orchestrator{config, nodeMap, srvMap, make(map[string]*StatusDetail)}, nil
}

func (o *Orchestrator) Start() error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	type nodResponse struct {
		IsConnected bool
		Error       error
	}
	e.GET("/orchestrator/configuration", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.config)
	})
	e.GET("/orchestrator/configuration/services", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.config.Services)
	})
	e.GET("/orchestrator/configuration/nodes", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.config.Nodes)
	})
	e.GET("/orchestrator/configuration/services/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		srv, ok := o.services[name[0]]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.JSON(http.StatusOK, srv.ServiceConfiguration)
	})
	e.GET("/orchestrator/configuration/nodes/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		node, ok := o.nodes[name[0]]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.JSON(http.StatusOK, node.NodeConfiguration)
	})
	e.GET("/orchestrator/services", func(c echo.Context) error {
		return c.JSON(http.StatusOK, o.statusdetail)
	})
	e.GET("/orchestrator/services/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		d, ok := o.statusdetail[name[0]]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		return c.JSON(http.StatusOK, d)
	})
	e.GET("/orchestrator/nodes", func(c echo.Context) error {
		m := make(map[string]nodResponse)
		for nodName, node := range o.nodes {
			con, err := node.IsConnected()
			m[nodName] = nodResponse{con, err}
		}
		return c.JSON(http.StatusOK, m)
	})
	e.GET("/orchestrator/nodes/:name", func(c echo.Context) error {
		name := c.ParamValues()
		if len(name) != 1 {
			return c.NoContent(http.StatusBadRequest)
		}
		node, ok := o.nodes[name[0]]
		if !ok {
			return c.NoContent(http.StatusBadRequest)
		}
		con, err := node.IsConnected()
		return c.JSON(http.StatusOK, nodResponse{con, err})
	})
	go o.Go()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return e.Start(o.services[OrchestratorServiceName].URL)
}

func (o *Orchestrator) Go() {
	time.Sleep(time.Duration(3) * time.Second)
	c := make(chan Event, 100)
	for srvName, srv := range o.services {
		go func(srvName string, srv *Service, c chan Event) {
			for {
				sd := make(StatusDetail)
				for nodName, node := range srv.node {
					err := node.ServiceStatus(srvName)
					if err != nil {
						sd[nodName] = err.Error()
					} else {
						sd[nodName] = StatusActive
					}
				}
				c <- Event{srvName, sd}
				if srv.Timeout <= 0 {
					break
				}
				time.Sleep(time.Duration(srv.Timeout) * time.Second)
			}
		}(srvName, srv, c)
	}
	for {
		e := <-c
		o.statusdetail[e.ServiceName] = &e.Status
	}
}

// func (o *Orchestrator) Stop() error {
// 	return nil
// }
