package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/bep/debounce"
	"github.com/fsnotify/fsnotify"
	"github.com/google/shlex"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	watch_dirs := []string{}
	run_cmds := []string{}

	debounced := debounce.New(100 * time.Millisecond)
	cmd_exec := func() {
		for i, v := range run_cmds {
			args, err := shlex.Split(v)

			log.Printf("run cmd %d: %s", i, v)
			log.Printf("run cmd %d: %s", i, args)

			cmd := exec.Command(args[0], args[1:]...)
			stdoutStderr, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%s\n", stdoutStderr)
		}
	}

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) {
					log.Println("modified file:", event.Name)
					debounced(cmd_exec)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	args := os.Args

	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.Func("w", "watch dirs", func(s string) error {
		if _, err := os.Stat(s); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("path not found"))
		}

		watch_dirs = append(watch_dirs, s)
		return nil
	})

	fs.Func("c", "run commands while path changed", func(s string) error {
		run_cmds = append(run_cmds, s)
		return nil
	})

	err = fs.Parse(args[1:])
	if err != nil {
		return
	}

	// Add a path.
	for i, v := range watch_dirs {
		log.Printf("watch path %d: %s", i, v)
		err = watcher.Add(v)
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(watch_dirs) == 0 {
		fmt.Printf("Usage of %s\n", args[0]);
		fs.PrintDefaults()
		return
	}

	// Block main goroutine forever.
	<-make(chan struct{})
}
