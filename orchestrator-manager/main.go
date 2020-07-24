package main

import (
	"fmt"
	"log"
	"os"

	orch "github.com/mariiatuzovska/orchestrator"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = fmt.Sprintf("%s-manager", orch.ServiceName)
	app.Usage = "Is a management service for discovering local/remote services activity"
	app.Description = "API service for orchestrator's services management"
	app.Version = orch.Version
	app.Copyright = "2020, mariiatuzovska"
	app.Authors = []cli.Author{{Name: "Tuzovska Mariia"}}
	app.Commands = []cli.Command{
		{
			Name:        "s",
			Usage:       "Start",
			Aliases:     []string{"r", "run", "start"},
			Description: "Strats service",
			Action: func(c *cli.Context) error {
				services, err := orch.NewServiceConfigurationArray(c.String("s"))
				if err != nil {
					err = services.Save(c.String("s"))
					if err != nil {
						log.Fatal(err)
					}
				}
				nodes, err := orch.NewNodeConfigurationArray(c.String("n"))
				if err != nil {
					err = nodes.Save(c.String("n"))
					if err != nil {
						log.Fatal(err)
					}
				}
				o, err := orch.NewOrchestrator(&orch.Configuration{
					ServicesPath: c.String("s"),
					NodesPath:    c.String("n"),
					Services:     services,
					Nodes:        nodes,
				})
				if err != nil {
					log.Fatal(err)
				}
				return o.StartOrchestrator(c.String("host") + ":" + c.String("port"))
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "s",
					Usage: "Path to services configuration file",
					Value: "./service-configuration.json",
				},
				&cli.StringFlag{
					Name:  "n",
					Usage: "Path to nodes configuration file",
					Value: "./node-configuration.json",
				},
				&cli.StringFlag{
					Name:  "host",
					Usage: "Host",
					Value: "127.0.0.1",
				},
				&cli.StringFlag{
					Name:  "port",
					Usage: "Port",
					Value: "6000",
				},
			},
		},
	}
	app.Run(os.Args)
}
