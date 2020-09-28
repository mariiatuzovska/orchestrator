package orchestrator

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo"
)

type JSONMessage struct {
	Message string
}

type ServiceStatusInfoResponse struct {
	ServiceName string
	StatusInfo  ServiceStatusInfo
}

/*
GetServicesController - Returns services
@url /orchestrator/services
@method GET
@response []Service
@response-type application/json
*/
func (o *Orchestrator) GetServicesController(c echo.Context) error {
	m := make([]*Service, 0)
	for _, service := range o.service {
		m = append(m, service)
	}
	return c.JSON(http.StatusOK, m)
}

/*
GetServiceByNameController - Returns service by ServiceName
@url /orchestrator/services/<ServiceName>
@method GET
@response Service
@response-type application/json
*/
func (o *Orchestrator) GetServiceByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srvInfo, ok := o.service[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	return c.JSON(http.StatusOK, srvInfo)
}

/*
StartServiceByNameController - Starts service
@url /orchestrator/services/<ServiceName>/<NodeName>
@method POST
@response-type text/plain
*/
func (o *Orchestrator) StartServiceByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 2 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srv, ok := o.service[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	node, ok := o.node[name[1]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
	}
	ok = false
	for _, n := range srv.Nodes {
		if n.NodeName == node.NodeName {
			ok = true
			break
		}
	}
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("%s service has not %s node", srv.ServiceName, node.NodeName),
		})
	}
	err := o.StartService(node.NodeName, srv.ServiceName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{
			fmt.Sprintf("Orchestrator: %s.%s starting error: %s", srv.ServiceName, node.NodeName, err.Error()),
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
func (o *Orchestrator) StopServiceByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 2 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srv, ok := o.service[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	node, ok := o.node[name[1]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
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
	err := o.StopService(node.NodeName, srv.ServiceName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{
			fmt.Sprintf("Orchestrator: %s.%s starting error: %s", srv.ServiceName, node.NodeName, err.Error()),
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
func (o *Orchestrator) GetNodesController(c echo.Context) error {
	m := make([]*Node, 0)
	for _, node := range o.node {
		m = append(m, node)
	}
	return c.JSON(http.StatusOK, m)
}

/*
GetNodeByNameController - Returns node by NodeName
@url /orchestrator/nodes/<NodeName>
@method GET
@response Node
@response-type application/json
*/
func (o *Orchestrator) GetNodeByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	nodInfo, ok := o.node[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
	}
	return c.JSON(http.StatusOK, nodInfo)
}

/*
ConnectToNodeByNameController - Reconnects to node
@url /orchestrator/nodes/<NodeName>
@method POST
@response-type text/plain
*/
func (o *Orchestrator) ConnectToNodeByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	node, ok := o.node[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
	}
	if node.nodeStatus == StatusConnected {
		return c.NoContent(http.StatusNoContent)
	}
	o.mux.Lock()
	if node.Connection != nil {
		client, err := node.Connect()
		if err == nil {
			o.client[node.NodeName] = client
			o.node[node.NodeName].nodeStatus = StatusConnected
		} else {
			log.Printf("Orshestartor: %s node: %s\n", node.NodeName, err.Error())
			o.node[node.NodeName].nodeStatus = StatusDisconnected
		}
	} else {
		o.node[node.NodeName].nodeStatus = StatusConnected
	}
	o.mux.Unlock()
	return c.NoContent(http.StatusNoContent)
}

/*
GetServiceStatusesController - Returns services statuses
@url /orchestrator/statuses
@method GET
@response []ServiceStatusInfoResponse
@response-type application/json
*/
func (o *Orchestrator) GetServiceStatusesController(c echo.Context) error {
	statuses := make([]ServiceStatusInfoResponse, 0)
	for _, service := range o.service {
		statuses = append(statuses, ServiceStatusInfoResponse{service.ServiceName, service.serviceStatus})
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
func (o *Orchestrator) GetServiceStatusByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srvInfo, ok := o.service[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	return c.JSON(http.StatusOK, ServiceStatusInfoResponse{srvInfo.ServiceName, srvInfo.serviceStatus})
}
