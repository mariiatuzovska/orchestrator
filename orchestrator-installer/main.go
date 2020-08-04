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
			Name:        "install",
			Aliases:     []string{"i"},
			Usage:       "Installing service",
			Description: "Remote installation of service",
			Action: func(c *cli.Context) error {
				installer := orch.ServiceInstaller{
					ServicePackage: c.String("package"),
					OS:             c.String("os"),
					Connection: &orch.Connection{
						Host:       c.String("host"),
						Port:       c.String("port"),
						User:       c.String("user"),
						SSHKey:     c.String("key"),
						PassPhrase: c.String("ph"),
					},
				}
				return installer.InstallService()
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
					Name:  "ph",
					Usage: "PassPhrase",
				},
			},
		},
		{
			Name:        "copy",
			Aliases:     []string{"c", "cp"},
			Usage:       "Copying file",
			Description: "Remote copying of file",
			Action: func(c *cli.Context) error {
				fs := orch.FileSetter{
					Path: c.String("path"),
					OS:   c.String("os"),
					Connection: &orch.Connection{
						Host:       c.String("host"),
						Port:       c.String("port"),
						User:       c.String("user"),
						SSHKey:     c.String("key"),
						PassPhrase: c.String("ph"),
					},
				}
				if c.String("file") == "" {
					log.Println("file path is undefined")
					os.Exit(1)
				}
				file, err := os.Open(c.String("file"))
				if err != nil {
					return err
				}
				defer file.Close()
				return fs.SetFile(file)
			},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "file,f",
					Usage: "Path to file",
					Value: "",
				},
				&cli.StringFlag{
					Name:  "path,p",
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
					Name:  "ph",
					Usage: "PassPhrase",
				},
			},
		},
	}
	app.Run(os.Args)
}
