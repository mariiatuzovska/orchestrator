package orchestrator

type Service struct {
	DNS   string
	Nodes Nodes
}

// ServiceName is a unique value for orchestrators configuration file.
// Defines service name
type ServiceName string

type ServiceStatus struct {
	DNS   string
	Nodes *NodeStatus
}
