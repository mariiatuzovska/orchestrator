package orchestrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type NodeConfigurationArray []*NodeConfiguration

type ServiceConfigurationArray []*ServiceConfiguration

// NewServiceConfigurationArray creates new ServiceConfigurationArray by path or takes default if one does not exist
func NewServiceConfigurationArray(configPath string) (*ServiceConfigurationArray, error) {
	var file []byte
	config := new(ServiceConfigurationArray)
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration file not found, initializing with default\n")
		config = DefaultServiceConfigurationArray()
		config.Save(configPath)
		return config, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration file is broken, initializing with default\n")
		config = DefaultServiceConfigurationArray()
		config.Save(configPath)
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
		fmt.Fprintf(os.Stderr, "Configuration file not found, initializing with default\n")
		config = DefaultNodeConfigurationArray()
		config.Save(configPath)
		return config, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration file is broken, initializing with default\n")
		config = DefaultNodeConfigurationArray()
		config.Save(configPath)
		return config, err
	}
	return config, nil
}

// DefaultNodeConfigurationArray returns default NodeConfigurationArray
func DefaultNodeConfigurationArray() *NodeConfigurationArray {
	def := &NodeConfigurationArray{
		{
			NodeName: "this",
			OS:       "darwin",
		},
		{
			NodeName: "test",
			OS:       "linux",
			Connection: &Connection{
				Host:   "0.0.0.0",
				User:   "root",
				SSHKey: "~/.ssh/my_key",
			},
		},
	}
	return def
}

// DefaultServiceConfigurationArray returns default ServiceConfigurationArray
func DefaultServiceConfigurationArray() *ServiceConfigurationArray {
	def := &ServiceConfigurationArray{
		{
			ServiceInfo{
				ServiceName: "com.orchestrator.app",
				ServiceType: "orchestrator",
				URL:         "localhost:6000",
				HTTPAccess: []*HTTPAccess{
					{
						Address:    "http://localhost:6000/orchestrator/nodes",
						Method:     "GET",
						StatusCode: 200,
					},
					{
						Address:    "http://localhost:6000/orchestrator/services",
						Method:     "GET",
						StatusCode: 200,
					},
				},
				Timeout: 10,
			},
			[]string{"this"},
		},
		{
			ServiceInfo{
				ServiceName: "myuser",
				ServiceType: "user",
				URL:         "http://myuser.com.ua",
				HTTPAccess: []*HTTPAccess{
					{
						Address:    "http://myuser.com.ua/api/v1/users",
						Method:     "GET",
						StatusCode: 200,
					},
				},
				Timeout: 10,
			},
			[]string{"test"},
		},
	}
	return def
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
