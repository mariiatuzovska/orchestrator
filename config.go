package orchestrator

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
			DNS: "orchestrator.com.ua",
			Nodes: Nodes{
				"orchestartor_1": {
					Romote:           false,
					StartImmediately: false,
					OS:               "darwin",
					HTTPAccess: []HTTPAccess{
						{
							Address:    "http://orchestrator.com.ua/orchestrator/status",
							Method:     "GET",
							StatusCode: 200,
						},
						{
							Address:    "http://orchestrator.com.ua/orchestrator/service",
							Method:     "GET",
							StatusCode: 200,
						},
					},
					Commands: Commands{
						"start":  "launchctl load ~/Library/LaunchAgents/com.orchestrator.app.plist",
						"stop":   "launchctl unload ~/Library/LaunchAgents/com.orchestrator.app.plist",
						"status": "launchctl list | grep com.orchestrator.app",
					},
					Settings: Settings{
						Timeout: 30,
						StatusCommands: Commands{
							"status": "-\t0\tcom.orchestrator.app\n",
						},
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
