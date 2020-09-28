package orchestrator

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type FileSetter struct {
	Path       string
	OS         string
	Connection *Connection
}

func (fs *FileSetter) SetFile(file *os.File) error {
	if err := fs.Connection.Valid(); err != nil {
		return err
	}
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	fileCopyName, globalPath, err := parsePath(fs.OS, fs.Path)
	if err != nil {
		return err
	}
	log.Printf("Orchestrator file-setter: copying file into %s:%s %s\n", fs.Connection.Host, fs.Connection.Port, fs.Path)
	client, err := fs.Connection.connect()
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
	switch fs.OS {
	case OSLinux, OSDarwin:
		err = session.Run(fmt.Sprintf("/usr/bin/scp -t %s", globalPath)) // installing from `/tmp` for linux & darwin
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Orchestrator file-setter: installing is not provided for %s OS", OSWindows)
	}
	err = <-errChan
	if err != nil {
		log.Printf("Orchestrator file-setter: %s\n", err.Error())
		return err
	}
	log.Printf("Orchestrator file-setter: `%s` file has been successfully copied into %s:%s %s",
		fileCopyName, fs.Connection.Host, fs.Connection.Port, globalPath)
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
