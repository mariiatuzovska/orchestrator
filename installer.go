package orchestrator

import (
	"fmt"
	"log"
	"os"
	"path"
)

type ServiceInstaller struct {
	ServicePackage string
	OS             string
	Connection     *Connection
}

func (t *ServiceInstaller) InstallService() error {
	file, err := os.Open(t.ServicePackage)
	if err != nil {
		return err
	}
	defer file.Close()
	fileCopyName, _, err := parsePath(t.OS, t.ServicePackage)
	if err != nil {
		return err
	}
	fs := new(FileSetter)
	switch t.OS {
	case OSLinux, OSDarwin:
		fs = &FileSetter{path.Join("/tmp", fileCopyName), t.OS, t.Connection}
	default:
		return fmt.Errorf("Orchestrator service-installer: installing is not provided for %s OS", t.OS)
	}
	err = fs.SetFile(file)
	if err != nil {
		return err
	}
	client, err := t.Connection.connect()
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	// installing
	switch t.OS {
	case OSLinux, OSDarwin:
		command := fmt.Sprintf(LinuxInstallingDebFormatString, path.Join("/tmp", fileCopyName))
		out, err := session.CombinedOutput(command)
		if err != nil {
			return err
		}
		log.Printf("Orchestrator service-installer: %s", out)
	default:
		return fmt.Errorf("Orchestrator service-installer: installing is not provided for %s OS", t.OS)
	}
	return nil
}
