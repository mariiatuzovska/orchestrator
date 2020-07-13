package orchestrator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Configuration struct {
	Nodes    []*NodeConfiguration
	Services []*ServiceConfiguration
}

// NewConfiguration creates new Configuration by path or takes default if one does not exist
func NewConfiguration(configPath string) (*Configuration, error) {
	var file []byte
	config := new(Configuration)
	file, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration file not found, initializing with default\n")
		config = DefaultConfiguration()
		config.Save(configPath)
		return config, err
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration file is broken, initializing with default\n")
		config = DefaultConfiguration()
		config.Save(configPath)
		return config, err
	}
	return config, nil
}

// DefaultConfiguration returns default Configuration
func DefaultConfiguration() *Configuration {
	def := &Configuration{
		Nodes: []*NodeConfiguration{
			{
				NodeName:         "this",
				OS:               "darwin",
				StartImmediately: true,
				Remote:           false,
			},
			{
				NodeName:         "test",
				OS:               "linux",
				StartImmediately: true,
				Remote:           true,
				Connection: &Connection{
					Host:   "0.0.0.0",
					User:   "root",
					SSHKey: "~/.ssh/my_key",
				},
			},
		},
		Services: []*ServiceConfiguration{
			{
				ServiceName: "orchestrator",
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
				Nodes: []string{"this"},
			},
			{
				ServiceName: "myuser",
				URL:         "http://myuser.com.ua",
				HTTPAccess: []*HTTPAccess{
					{
						Address:    "http://myuser.com.ua/api/v1/users",
						Method:     "GET",
						StatusCode: 200,
					},
				},
				Nodes: []string{"test"},
			},
		},
	}
	return def
}

func (config *Configuration) Save(configPath string) error {
	a, err := json.MarshalIndent(config, "", "	")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, a, os.ModePerm)
}
