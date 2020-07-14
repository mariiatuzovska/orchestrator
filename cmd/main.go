package main

import (
	"log"
	"os"

	orch "github.com/mariiatuzovska/orchestrator"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "orchestrator"
	app.Version = orch.Version
	app.Copyright = "2020, mariiatuzovska"
	app.Authors = []cli.Author{{Name: "Tuzovska Mariia"}}
	app.Commands = []cli.Command{
		{
			Name:        "s",
			Usage:       "Start",
			Description: "Strats service",
			Action: func(c *cli.Context) error {
				services, err := orch.NewServiceConfigurationArray(c.String("sc"))
				if err != nil {
					log.Fatal(err)
				}
				nodes, err := orch.NewNodeConfigurationArray(c.String("nc"))
				if err != nil {
					log.Fatal(err)
				}
				o, err := orch.NewOrchestrator(&orch.Configuration{
					c.String("sc"), c.String("nc"), services, nodes,
				})
				if err != nil {
					log.Fatal(err)
				}
				return o.StartOrchestrator(c.String("host") + ":" + c.String("port"))
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "sc",
					Usage: "Path to services configuration file",
					Value: "./service-configuration.json",
				},
				&cli.StringFlag{
					Name:  "nc",
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
