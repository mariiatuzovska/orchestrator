package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	orch "github.com/mariiatuzovska/orchestrator"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = fmt.Sprintf("%s-installer", orch.ServiceName)
	app.Usage = "Is a cmd application for remote service installation"
	app.Version = orch.Version
	app.Copyright = "2020, mariiatuzovska"
	app.Authors = []cli.Author{{Name: "Tuzovska Mariia"}}
	app.Commands = []cli.Command{
		{
			Name:        "i",
			Usage:       "Install service",
			Description: "Remote installation of service",
			Action: func(c *cli.Context) error {
				t := orch.ServiceTemplate{
					c.String("package"),
					c.String("os"),
					orch.Connection{
						c.String("host"),
						c.String("port"),
						c.String("user"),
						c.String("key"),
						c.String("f"),
					},
				}
				err := t.InstallService()
				if err != nil {
					log.Println(err)
				}
				return err
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "package,p",
					Usage: "Path to pacakage",
					Value: "./",
				},
				&cli.StringFlag{
					Name:  "os,o",
					Usage: "OS",
					Value: runtime.GOOS,
				},
				&cli.StringFlag{
					Name:  "host",
					Usage: "Host",
				},
				&cli.StringFlag{
					Name:  "port",
					Usage: "Port",
					Value: "22",
				},
				&cli.StringFlag{
					Name:  "user,u",
					Usage: "User",
					Value: "root",
				},
				&cli.StringFlag{
					Name:  "key,ssh,k",
					Usage: "SSH key path",
					Value: "~/.ssh/id_rsa",
				},
				&cli.StringFlag{
					Name:  "f",
					Usage: "PassPhrase",
				},
			},
		},
	}
	app.Run(os.Args)
}
