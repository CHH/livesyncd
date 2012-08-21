package main

import (
	"flag"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Start forwarding events from the watcher's channel to the
// Sync Backend implementation. Quits when true is sent to the
// "quit" channel.
func startWatchLoop(sync chan *SyncEvent, quit chan bool, watcher *fsnotify.Watcher) {
	go func() {
		reIndexTick := time.NewTicker(60 * time.Second)

		for {
			select {
			case ev := <-watcher.Event:
				log.Printf("Received event: %+v", ev)

				go func() {
					if isExcluded(ev.Name) {
						return
					}

					syncEv := &SyncEvent{Name: ev.Name, FileEvent: ev}

					switch {
					case ev.IsCreate(), ev.IsModify():
						syncEv.Type = SYNC_PUT

					// Delete the file, regardless if it was deleted or renamed.
					// When it was renamed, then the uploading of the new file
					// will already be catched by an CREATE event.
					case ev.IsDelete(), ev.IsRename():
						syncEv.Type = SYNC_DELETE
					}

					// Notify the sync backend.
					sync <- syncEv
				}()
			case <-reIndexTick.C:
				log.Println("Refreshing Index")
				go refreshIndex(root, watcher, sync)

			case err := <-watcher.Error:
				log.Printf("Received error: %s", err)

			case <-quit:
				return
			}
		}
	}()
}

// Adds all sub directories of the given directory to the watcher
// and returns the number of watched directories.
func addWatchesRecursive(dir string, watcher *fsnotify.Watcher) int {
	var watched int
	entries, err := ioutil.ReadDir(dir)

	if err != nil {
		log.Println(err)
		return 0
	}

	watcher.Watch(dir)
	index[dir] = true
	watched++

	for _, e := range entries {
		name := e.Name()

		if e.IsDir() && !strings.HasPrefix(name, ".") {
			watched += addWatchesRecursive(filepath.Join(dir, name), watcher)
		}
	}

	return watched
}

func refreshIndex(dir string, watcher *fsnotify.Watcher, sync chan *SyncEvent) {
	entries, err := ioutil.ReadDir(dir)

	if err != nil {
		log.Println(err)
		return
	}

	_, inIndex := index[dir]

	if !inIndex {
		watcher.Watch(dir)
	}

	index[dir] = true

	for _, e := range entries {
		name := e.Name()

		if e.IsDir() && !strings.HasPrefix(name, ".") {
			refreshIndex(filepath.Join(dir, name), watcher, sync)

			// Schedule the file for syncing if the directory was not found
			// within the index
		} else if !inIndex {
			sync <- &SyncEvent{Name: filepath.Join(dir, name), Type: SYNC_PUT}
		}
	}
}

func isExcluded(name string) bool {
	for _, p := range config.Ignore {
		rel, _ := filepath.Rel(root, name)
		matched, err := filepath.Match(p, rel)

		if err != nil {
			log.Printf("Error in Pattern %q: %s", p, err)
		}

		if matched {
			log.Printf("Ignore: %q matched pattern %q", name, p)
			return true
		}
	}

	return false
}

// Daemon Configuration
type Config struct {
	RemoteHost string
	RemoteRoot string
	Ignore     []string
}

var (
	// Root of the directory to watch
	root string

	// Configuration
	config Config

	// Index of all watched directories (key: path)
	index map[string]bool
)

func init() {
	var ignore string

	flag.StringVar(&config.RemoteHost, "remote-host", "", "Remote Host Name ([user@]host)")
	flag.StringVar(&config.RemoteRoot, "remote-root", "", "Remote Root")
	flag.StringVar(&ignore, "ignore", "", "Comma delimited set of ignore patterns")

	flag.Parse()

	if config.RemoteHost == "" {
		log.Fatalln("Missing --remote-host")
	}

	if config.RemoteRoot == "" {
		log.Fatalln("Missing --remote-root")
	}

	index = make(map[string]bool)

	config.Ignore = strings.Split(ignore, ",")

	root, _ = os.Getwd()
}

func main() {
	log.Println("livesyncd running")
	log.Println("Stop with [CTRL] + [c]")
	log.Println("Ignore: ", config.Ignore)

	rlimit := new(syscall.Rlimit)
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, rlimit)

	rlimit.Cur = rlimit.Max

	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, rlimit)

	if err != nil {
		log.Panicf("Could not change Rlimit: %q", err)
	}

	watcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Panicln(err)
	}

	events := make(chan *SyncEvent)
	quitSync := make(chan bool)

	quitWatcher := make(chan bool)

	sigInt := make(chan os.Signal)
	signal.Notify(sigInt, os.Interrupt)

	StartSFTPSync(events, quitSync, config)
	startWatchLoop(events, quitWatcher, watcher)

	watched := addWatchesRecursive(root, watcher)

	log.Printf("Found %d directories to watch\n", watched)

	select {
	case <-sigInt:
		log.Println("Stopping to watch...")

		// Wait until the watcher has finished quitting
		quitWatcher <- true
		log.Println("Done")

		// Close all file handles, opened by the watcher
		watcher.Close()

		log.Println("Stopping Sync Backend...")
		quitSync <- true
		<-quitSync
		log.Println("Done")

		return
	}
}
