package orchestrator

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type JSONMessage struct {
	Message string
}

type ServiceStatusInfoResponse struct {
	ServiceName string
	StatusInfo  ServiceStatusInfo
}

type Server struct {
	*echo.Echo
	Orchestrator *Orchestrator
}

type BodyWithPassPhrase struct {
	PassPhrase string
}

func (o *Orchestrator) Server() *Server {
	s := &Server{echo.New(), o}
	s.HideBanner = true
	s.HidePort = true
	// INFO
	s.GET("/orchestrator", func(c echo.Context) error { return c.JSON(http.StatusOK, s.Routes()) })
	s.GET("/orchestrator/services", s.GetServicesController)
	// SERVICES
	s.GET("/orchestrator/services", s.GetServicesController)
	s.GET("/orchestrator/services/:ServiceName", s.GetServiceByNameController)
	// SERVICES: START / STOP
	s.POST("/orchestrator/services/:ServiceName/:NodeName", s.StartServiceByNameController)
	s.DELETE("/orchestrator/services/:ServiceName/:NodeName", s.StopServiceByNameController)
	// NODES
	s.GET("/orchestrator/nodes", s.GetNodesController)
	s.GET("/orchestrator/nodes/:NodeName", s.GetNodeByNameController)
	s.POST("/orchestrator/nodes/:NodeName", s.ConnectToNodeByNameController)
	s.DELETE("/orchestrator/nodes/:NodeName", s.DisconnectNodeByNameController)
	// STATUSES
	s.GET("/orchestrator/statuses", s.GetServiceStatusesController)
	s.GET("/orchestrator/statuses/:ServiceName", s.GetServiceStatusByNameController)

	s.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		ExposeHeaders:    []string{"Server", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
	}))
	return s
}

/*
GetServicesController - Returns services
@url /orchestrator/services
@method GET
@response []Service
@response-type application/json
*/
func (s *Server) GetServicesController(c echo.Context) error {
	return c.JSON(http.StatusOK, s.Orchestrator.copyServicesAsArray())
}

/*
GetServiceByNameController - Returns service by ServiceName
@url /orchestrator/services/<ServiceName>
@method GET
@response Service
@response-type application/json
*/
func (s *Server) GetServiceByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameter"})
	}
	srv, err := s.Orchestrator.GetService(name[0])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	return c.JSON(http.StatusOK, srv)
}

/*
StartServiceByNameController - Starts service
@url /orchestrator/services/<ServiceName>/<NodeName>
@method POST
@response-type text/plain
*/
func (s *Server) StartServiceByNameController(c echo.Context) error {
	param := c.ParamValues()
	if len(param) != 2 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameters"})
	}
	if err := s.Orchestrator.StartService(param[1], param[0]); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{
			fmt.Sprintf("Orchestrator: %s.%s starting error: %s", param[0], param[1], err.Error()),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

/*
StopServiceByNameController - Stops service
@url /orchestrator/services/<ServiceName>/<NodeName>
@method DELETE
@response-type text/plain
*/
func (s *Server) StopServiceByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 2 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameters"})
	}
	srv, err := s.Orchestrator.GetService(name[0])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	node, err := s.Orchestrator.GetNode(name[1])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	exist := false
	for _, n := range srv.Nodes {
		if n.NodeName == node.NodeName {
			exist = true
			break
		}
	}
	if !exist {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("%s service has no node with name %s", srv.ServiceName, node.NodeName),
		})
	}
	if err := s.Orchestrator.StopService(node.NodeName, srv.ServiceName); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{
			fmt.Sprintf("Orchestrator: %s.%s stopping error: %s", srv.ServiceName, node.NodeName, err.Error()),
		})
	}
	return c.NoContent(http.StatusNoContent)
}

/*
GetNodesController - Returns nodes
@url /orchestrator/nodes
@method GET
@response []Node
@response-type application/json
*/
func (s *Server) GetNodesController(c echo.Context) error {
	return c.JSON(http.StatusOK, s.Orchestrator.copyNodesAsArray())
}

/*
GetNodeByNameController - Returns node by NodeName
@url /orchestrator/nodes/<NodeName>
@method GET
@response Node
@response-type application/json
*/
func (s *Server) GetNodeByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameter"})
	}
	node, err := s.Orchestrator.GetNode(name[0])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	return c.JSON(http.StatusOK, node)
}

/*
ConnectToNodeByNameController - Reconnects to node
@url /orchestrator/nodes/<NodeName>
@method POST
@request BodyWithPassPhrase
@response-type text/plain
*/
func (s *Server) ConnectToNodeByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameter"})
	}
	body := BodyWithPassPhrase{}
	c.Bind(&body)
	node, err := s.Orchestrator.GetNode(name[0])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	if node.NodeStatus == StatusConnected {
		return c.NoContent(http.StatusNoContent)
	}
	if err := s.Orchestrator.ConnectNode(name[0], body.PassPhrase); err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

/*
DisconnectNodeByNameController - Reconnects to node
@url /orchestrator/nodes/<NodeName>
@method DELETE
@response-type text/plain
*/
func (s *Server) DisconnectNodeByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameter"})
	}
	node, err := s.Orchestrator.GetNode(name[0])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	if node.NodeStatus == StatusDisconnected {
		return c.NoContent(http.StatusNoContent)
	}
	if err := s.Orchestrator.DisconnectNode(name[0]); err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

/*
GetServiceStatusesController - Returns services statuses
@url /orchestrator/statuses
@method GET
@response []ServiceStatusInfoResponse
@response-type application/json
*/
func (s *Server) GetServiceStatusesController(c echo.Context) error {
	statuses := make([]ServiceStatusInfoResponse, 0)
	services := s.Orchestrator.copyServicesAsArray()
	for _, service := range services {
		statuses = append(statuses, ServiceStatusInfoResponse{service.ServiceName, service.ServiceStatus})
	}
	return c.JSON(http.StatusOK, statuses)
}

/*
GetServiceStatusByNameController - Returns services status by ServiceName
@url /orchestrator/statuses/<ServiceName>
@method GET
@response ServiceStatusInfoResponse
@response-type application/json
*/
func (s *Server) GetServiceStatusByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind url parameter"})
	}
	srv, err := s.Orchestrator.GetService(name[0])
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	return c.JSON(http.StatusOK, ServiceStatusInfoResponse{srv.ServiceName, srv.ServiceStatus})
}
