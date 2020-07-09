package orchestrator

type Service struct {
	URL   string
	Nodes Nodes
}

// ServiceName is a unique value for orchestrators configuration file.
// Defines service name
type ServiceName string

type ServiceStatus struct {
	URL          string
	NodeStatuses map[NodeName]StatusValue
}
