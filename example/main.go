package main

import (
	"log"

	"github.com/mariiatuzovska/orchestrator"
)

func main() {

	node1 := orchestrator.NewNode(&orchestrator.NodeInfo{
		NodeName: "server",
		OS:       orchestrator.OSLinux,
		Connection: &orchestrator.Connection{
			Host:   "172.16.0.105",
			User:   "mariiatuzovska",
			SSHKey: "~/.ssh/id_rsa",
		},
	})

	node2 := orchestrator.NewNode(&orchestrator.NodeInfo{
		NodeName: "local",
		OS:       orchestrator.OSDarwin,
	})

	service := orchestrator.NewService(&orchestrator.ServiceInfo{
		ServiceName: "myservice.service",
		URL:         "172.16.0.105:8080",
		HTTPAccess: []*orchestrator.HTTPAccess{
			{
				Method:     "GET",
				Address:    "http://172.16.0.105:8080/",
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
		TimeoutSeconds: 30,
	}, node1)

	orch := orchestrator.NewOrchestrator()
	go orch.Start()

	if err := orch.RegistrateNodes(node1, node2); err != nil {
		log.Fatal(err)
	}
	if err := orch.RegistrateServices(service); err != nil {
		log.Fatal(err)
	}

	orch.SetLogLevel(orchestrator.INFO)

	if err := orch.Server().Start("127.0.0.1:8080"); err != nil {
		log.Fatal(err)
	}

}
