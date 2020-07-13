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
				config, err := orch.NewConfiguration(c.String("config"))
				if err != nil {
					log.Fatal(err)
				}
				o, err := orch.NewOrchestrator(config)
				if err != nil {
					log.Fatal(err)
				}
				return o.Start()
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "config",
					Usage: "Path to configuration file",
					Value: "./config.json",
				},
			},
		},
	}
	app.Run(os.Args)
}
