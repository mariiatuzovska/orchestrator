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
	for _, service := range o.services {
		m = append(m, service)
	}
	return c.JSON(http.StatusOK, m)
}

/*
CreateServiceController - Creates new launched (started) service
@url /orchestrator/services
@method POST
@request ServiceConfiguration
@response Service
@response-type application/json
*/
func (o *Orchestrator) CreateServiceController(c echo.Context) error {
	srvConfig := new(ServiceConfiguration)
	if err := c.Bind(srvConfig); err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srv, err := NewService(srvConfig, o.nodes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	if _, exist := o.services[srv.ServiceName]; exist {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("Orchestrator: %s service is not unique / already defined / duplicated in ServiceConfigurationArray", srvConfig.ServiceName),
		})
	}
	o.services[srv.ServiceName] = srv
	if err = o.UpdateConfiguration(); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{err.Error()})
	}
	go o.ServiceStatusRoutine(srv)
	return c.JSON(http.StatusCreated, srv)
}

/*
UpdateServiceController - Updates launched service and reloads it
@url /orchestrator/services
@method PUT
@request ServiceConfiguration
@response Service
@response-type application/json
*/
func (o *Orchestrator) UpdateServiceController(c echo.Context) error {
	srvConfig := new(ServiceConfiguration)
	if err := c.Bind(srvConfig); err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srv, err := NewService(srvConfig, o.nodes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	if _, exist := o.services[srv.ServiceName]; !exist {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("Orchestrator: %s service not found", srvConfig.ServiceName),
		})
	}
	o.mux.Lock()
	o.services[srv.ServiceName] = srv
	o.mux.Unlock()
	if err = o.UpdateConfiguration(); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{err.Error()})
	}
	return c.JSON(http.StatusOK, srv)
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
	srvInfo, ok := o.services[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	return c.JSON(http.StatusOK, srvInfo)
}

/*
DeleteServiceController - Deletes launched service
@url /orchestrator/services/<ServiceName>
@method DELETE
@response-type text/plain
*/
func (o *Orchestrator) DeleteServiceController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	srv, exist := o.services[name[0]]
	if !exist {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("Orchestrator: %s service not found", name[0]),
		})
	}
	o.mux.Lock()
	delete(o.services, srv.ServiceName) // services routine will stops gracefully
	o.mux.Unlock()
	if err := o.UpdateConfiguration(); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
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
	srv, ok := o.services[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	node, ok := o.nodes[name[1]]
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
	srv, ok := o.services[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	node, ok := o.nodes[name[1]]
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
	for _, node := range o.nodes {
		m = append(m, node)
	}
	return c.JSON(http.StatusOK, m)
}

/*
CreateNodeController - Creates new launched (connected) node
@url /orchestrator/nodes
@method POST
@request NodeConfiguration
@response Node
@response-type application/json
*/
func (o *Orchestrator) CreateNodeController(c echo.Context) error {
	nodConfig := new(NodeConfiguration)
	if err := c.Bind(nodConfig); err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	nod, err := NewNode(nodConfig)
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	if _, exist := o.nodes[nodConfig.NodeName]; exist {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("Orchestrator: %s node is not unique / already defined / duplicated in NodeConfigurationArray", nodConfig.NodeName),
		})
	}
	o.nodes[nod.NodeName] = nod
	if err = o.UpdateConfiguration(); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{err.Error()})
	}
	if o.nodes[nod.NodeName].Connection != nil {
		client, err := o.nodes[nod.NodeName].Connect()
		if err == nil {
			o.remote[nod.NodeName] = client
			o.nodes[nod.NodeName].NodeStatus = StatusConnected
		} else {
			log.Printf("Orshestartor: %s node: %s\n", nod.NodeName, err.Error())
			o.nodes[nod.NodeName].NodeStatus = StatusDisconnected
		}
	} else {
		o.nodes[nod.NodeName].NodeStatus = StatusConnected
	}
	return c.JSON(http.StatusCreated, nod)
}

/*
UpdateNodeController - Updates node
@url /orchestrator/nodes
@method PUT
@request NodeConfiguration
@response Node
@response-type application/json
*/
func (o *Orchestrator) UpdateNodeController(c echo.Context) error {
	nodConfig := new(NodeConfiguration)
	if err := c.Bind(nodConfig); err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	nod, err := NewNode(nodConfig)
	if err != nil {
		return c.JSON(http.StatusBadRequest, JSONMessage{err.Error()})
	}
	if _, exist := o.nodes[nodConfig.NodeName]; !exist {
		return c.JSON(http.StatusBadRequest, JSONMessage{
			fmt.Sprintf("Orchestrator: %s node is not nound", nodConfig.NodeName),
		})
	}
	o.mux.Lock()
	o.nodes[nod.NodeName] = nod
	if _, ok := o.remote[nod.NodeName]; ok {
		err := o.remote[nod.NodeName].Close()
		if err != nil {
			log.Printf("Orshestartor: %s node: %s\n", nod.NodeName, err.Error())
			delete(o.remote, nod.NodeName)
		}
	}
	if o.nodes[nod.NodeName].Connection != nil {
		client, err := o.nodes[nod.NodeName].Connect()
		if err == nil {
			o.remote[nod.NodeName] = client
			o.nodes[nod.NodeName].NodeStatus = StatusConnected
		} else {
			log.Printf("Orshestartor: %s node: %s\n", nod.NodeName, err.Error())
			o.nodes[nod.NodeName].NodeStatus = StatusDisconnected
		}
	} else {
		o.nodes[nod.NodeName].NodeStatus = StatusConnected
	}
	o.mux.Unlock()
	if err = o.UpdateConfiguration(); err != nil {
		return c.JSON(http.StatusInternalServerError, JSONMessage{err.Error()})
	}
	return c.JSON(http.StatusCreated, nod)
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
	nodInfo, ok := o.nodes[name[0]]
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
	node, ok := o.nodes[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
	}
	if node.NodeStatus == StatusConnected {
		return c.NoContent(http.StatusNoContent)
	}
	o.mux.Lock()
	if node.Connection != nil {
		client, err := node.Connect()
		if err == nil {
			o.remote[node.NodeName] = client
			o.nodes[node.NodeName].NodeStatus = StatusConnected
		} else {
			log.Printf("Orshestartor: %s node: %s\n", node.NodeName, err.Error())
			o.nodes[node.NodeName].NodeStatus = StatusDisconnected
		}
	} else {
		o.nodes[node.NodeName].NodeStatus = StatusConnected
	}
	o.mux.Unlock()
	return c.NoContent(http.StatusNoContent)
}

/*
DeleteNodeByNameController - Deletes node by NodeName
@url /orchestrator/nodes/<NodeName>
@method DELETE
@response-type text/plain
*/
func (o *Orchestrator) DeleteNodeByNameController(c echo.Context) error {
	name := c.ParamValues()
	if len(name) != 1 {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Can't bind request"})
	}
	nodInfo, ok := o.nodes[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown node"})
	}
	o.mux.Lock()
	for _, srv := range o.services {
		for _, node := range srv.Nodes {
			if node.NodeName == nodInfo.NodeName {
				o.mux.Unlock()
				return c.JSON(http.StatusForbidden, JSONMessage{
					fmt.Sprintf("%s node is already using by %s service", node.NodeName, srv.ServiceName),
				})
			}
		}
	}
	delete(o.nodes, name[0])
	delete(o.remote, name[0])
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
	for _, service := range o.services {
		statuses = append(statuses, ServiceStatusInfoResponse{service.ServiceName, service.StatusInfo})
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
	srvInfo, ok := o.services[name[0]]
	if !ok {
		return c.JSON(http.StatusBadRequest, JSONMessage{"Unknown service"})
	}
	return c.JSON(http.StatusOK, ServiceStatusInfoResponse{srvInfo.ServiceName, srvInfo.StatusInfo})
}
