package main

import(
	"os/exec"
	"os"
	"fmt"
	"log"
	"path/filepath"
)

func StartSFTPSync(events chan *SyncEvent, quit chan bool, config Config) {
	sftp := exec.Command("sftp", "-C", config.RemoteHost+":"+config.RemoteRoot+"/")
	sftp.Stderr = os.Stderr
	sftp.Stdout = os.Stdout

	cmd, err := sftp.StdinPipe()

	if err != nil {
		log.Fatalf("Could not open Stdin of SFTP: %s", err)
	}

	if err := sftp.Start(); err != nil {
		log.Fatalf("Could not start SFTP process: %s", err)
	}

	go func() {
		for {
			select {
			case ev := <-events:
				switch {
				case ev.IsPut():
					remote, _ := filepath.Rel(root, ev.Name)
					remoteDir := filepath.Dir(remote)

					fmt.Fprintf(cmd, "! mkdir -p %q\n", remoteDir)
					fmt.Fprintf(cmd, "put -p %q %q\n", ev.Name, remote)
				case ev.IsDelete():
					remote, _ := filepath.Rel(root, ev.Name)

					fmt.Fprintf(cmd, "rm %q\n", remote)
				}
			case <-quit:
				fmt.Fprint(cmd, "exit\n")
				sftp.Wait()
				quit <- true
				return
			}
		}
	}()
}


