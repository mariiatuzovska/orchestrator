package orchestrator

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

func InstallDebianService(servicePath string, connection *Connection, passPhrase string) error {
	file, err := os.Open(servicePath)
	if err != nil {
		return err
	}
	defer file.Close()
	fileCopyName, _, err := parsePath(OSLinux, servicePath)
	if err != nil {
		return err
	}
	err = SetFileUnix(file, connection, path.Join("/tmp", fileCopyName), passPhrase)
	if err != nil {
		return err
	}
	client, err := connection.connect(passPhrase)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	command := fmt.Sprintf(LinuxInstallingDebFormatString, path.Join("/tmp", fileCopyName))
	out, err := session.CombinedOutput(command)
	if err != nil {
		return err
	}
	fmt.Printf("%s:%s running 'command': %s", connection.Host, connection.Port, string(out))
	return nil
}

func SetFileUnix(file *os.File, connection *Connection, path string, passPhrase string) error {
	if err := connection.Valid(); err != nil {
		return err
	}
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileCopyName, globalPath, err := parsePath(OSLinux, path)
	if err != nil {
		return err
	}
	client, err := connection.connect(passPhrase)
	if err != nil {
		return err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	errChan := make(chan error, 1)
	go func(errChan chan error) {
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
	}(errChan)
	err = session.Run(fmt.Sprintf("/usr/bin/scp -t %s", globalPath)) // installing from `/tmp` for linux & darwin
	if err != nil {
		return err
	}
	err = <-errChan
	if err != nil {
		return err
	}
	return nil
}

func parsePath(os, path string) (fileCopyName string, globalPath string, err error) {
	if os == OSLinux || os == OSDarwin {
		arrOfServicePackage := strings.Split(path, "/")
		if len(arrOfServicePackage) < 1 {
			return "", "", fmt.Errorf("Orchestrator: can't parse file path %s", path)
		}
		fileCopyName = arrOfServicePackage[len(arrOfServicePackage)-1]
		globalPath = strings.Join(arrOfServicePackage[:len(arrOfServicePackage)-1], "/")
	} else if os == OSWindows {
		arrOfServicePackage := strings.Split(path, `\`)
		fileCopyName = arrOfServicePackage[len(arrOfServicePackage)-1]
		globalPath = strings.Join(arrOfServicePackage[:len(arrOfServicePackage)-1], `\`)
	} else {
		return "", "", fmt.Errorf("Orchestrator: unknown OS")
	}
	if fileCopyName == "" {
		return "", "", fmt.Errorf("Orchestrator: can't parse file name for this path: %s", path)
	}
	return fileCopyName, globalPath, nil
}
