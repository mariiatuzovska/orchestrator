package orchestartor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Configuration map[ServiceName]*Service

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
		OrchestratorServiceName: {
			DNS: "https://orchestrator.com.ua",
			Nodes: []*Node{
				{
					Romote:           false,
					StartImmediately: false,
					OS:               "darwin",
					HTTPAccess: []HTTPAccess{
						{
							Address:    "https://orchestrator.com.ua/orchestrator/status",
							Method:     "GET",
							StatusCode: 200,
						},
						{
							Address:    "https://orchestrator.com.ua/orchestrator/service",
							Method:     "GET",
							StatusCode: 200,
						},
					},
					Commands: Commands{
						Start:  "launchctl load com.orchestrator.app.plist",
						Stop:   "launchctl unload com.orchestrator.app.plist",
						Status: "launchctl list | grep com.orchestrator.app",
					},
					Settings: Settings{
						Timeout: 30,
					},
				},
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

func (config *Configuration) Valid() bool {
	srv := *config
	_, ok := srv[OrchestratorServiceName]
	if !ok {
		return false
	}
	for _, node := range srv[OrchestratorServiceName].Nodes {
		if node.Romote {
			return false
		}
	}
	return true
}
