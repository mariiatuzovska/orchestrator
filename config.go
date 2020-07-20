package orchestrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
)

type NodeConfigurationArray []*NodeConfiguration

type ServiceConfigurationArray []*ServiceConfiguration

type Configuration struct {
	ServicesPath, NodesPath string
	Services                *ServiceConfigurationArray
	Nodes                   *NodeConfigurationArray
}

func (o *Orchestrator) UpdateConfiguration() error {
	nodes, services := make([]*NodeConfiguration, 0), make([]*ServiceConfiguration, 0)
	for _, node := range o.nodes {
		nodes = append(nodes, &node.NodeConfiguration)
	}
	for _, service := range o.services {
		n := make([]string, 0)
		for _, nod := range service.Nodes {
			n = append(n, nod.NodeName)
		}
		services = append(services, &ServiceConfiguration{service.ServiceInfo, n})
	}
	n, s := NodeConfigurationArray(nodes), ServiceConfigurationArray(services)
	o.config.Nodes, o.config.Services = &n, &s
	err := o.config.Nodes.Save(o.config.NodesPath)
	if err != nil {
		return err
	}
	err = o.config.Services.Save(o.config.ServicesPath)
	if err != nil {
		return err
	}
	return nil
}

// NewServiceConfigurationArray creates new ServiceConfigurationArray by path or takes default if one does not exist
func NewServiceConfigurationArray(configPath string) (*ServiceConfigurationArray, error) {
	var file []byte
	config := new(ServiceConfigurationArray)
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ServiceConfigurationArray configuration file not found, initializing with default\n")
		config = DefaultServiceConfigurationArray()
		return config, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ServiceConfigurationArray configuration file is broken, initializing with default\n")
		config = DefaultServiceConfigurationArray()
		return config, err
	}
	return config, nil
}

// NewNodeConfigurationArray creates new NodeConfigurationArray by path or takes default if one does not exist
func NewNodeConfigurationArray(configPath string) (*NodeConfigurationArray, error) {
	var file []byte
	config := new(NodeConfigurationArray)
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NodeConfigurationArray configuration file not found, initializing with default\n")
		config = DefaultNodeConfigurationArray()
		return config, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NodeConfigurationArray configuration file is broken, initializing with default\n")
		config = DefaultNodeConfigurationArray()
		return config, err
	}
	return config, nil
}

// DefaultNodeConfigurationArray returns default NodeConfigurationArray
func DefaultNodeConfigurationArray() *NodeConfigurationArray {
	return &NodeConfigurationArray{
		{
			NodeName: fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
			OS:       runtime.GOOS,
		},
	}
}

// DefaultServiceConfigurationArray returns default ServiceConfigurationArray
func DefaultServiceConfigurationArray() *ServiceConfigurationArray {
	return &ServiceConfigurationArray{}
}

func (config *NodeConfigurationArray) Save(configPath string) error {
	a, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, a, os.ModePerm)
}

func (config *ServiceConfigurationArray) Save(configPath string) error {
	a, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, a, os.ModePerm)
}
