package orchestrator

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

type ServiceTemplate struct {
	ServicePackage string
	OS             string
	Connection     Connection
}

func (t *ServiceTemplate) InstallService() error {
	file, err := os.Open(t.ServicePackage)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := t.Connection.valid(); err != nil {
		return err
	}
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileCopyName := ""
	if t.OS == OSLinux || t.OS == OSDarwin {
		arrOfServicePackage := strings.Split(t.ServicePackage, "/")
		fileCopyName = arrOfServicePackage[len(arrOfServicePackage)-1]
	} else if t.OS == OSWindows {
		// arrOfServicePackage := strings.Split(t.ServicePackage, `\`)
		// fileCopyName = arrOfServicePackage[len(arrOfServicePackage)-1]
		return fmt.Errorf("Orchestrator: copying is not provided for %s OS", t.OS)
	} else {
		return fmt.Errorf("Orchestrator: copying is not provided for unknown OS")
	}
	if fileCopyName == "" {
		return fmt.Errorf("Orchestrator: can't parse file name for %s ServicePackage", t.ServicePackage)
	}
	client, err := t.Connection.connect()
	if err != nil {
		return err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	// wg := sync.WaitGroup{}
	// wg.Add(1)
	errChan := make(chan error)
	go func() {
		hostIn, err := session.StdinPipe()
		if err != nil {
			errChan <- err
		}
		defer hostIn.Close()
		_, err = fmt.Fprintf(hostIn, "C0664 %d %s\n", stat.Size(), fileCopyName)
		if err != nil {
			errChan <- err
		}
		_, err = io.Copy(hostIn, file)
		if err != nil {
			errChan <- err
		}
		_, err = fmt.Fprint(hostIn, "\x00")
		if err != nil {
			errChan <- err
		}
		errChan <- nil
	}()
	switch t.OS {
	case OSWindows:
		return fmt.Errorf("Orchestrator: installing is not provided for %s OS", OSWindows)
	default:
		err = session.Run("/usr/bin/scp -t /tmp") // installing from `/tmp` for linux & darwin
		if err != nil {
			return err
		}
	}
	// waiting scp
	err = <-errChan // wg.Wait()
	if err != nil {
		return err
	}
	session, err = client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	log.Printf("Orchestrator: %s file has been copied as /tmp/%s", t.ServicePackage, fileCopyName)
	// installing
	switch t.OS {
	case OSLinux:
		command := fmt.Sprintf(LinuxInstallingDebFormatString, path.Join("/tmp", fileCopyName))
		out, err := session.CombinedOutput(command)
		if err != nil {
			return err
		}
		log.Printf("Orchestrator: %s", out)
	default:
		return fmt.Errorf("Orchestrator: installing is not provided for %s OS", t.OS)
	}
	return nil
}
