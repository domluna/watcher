// This just shows a basic way to setup
// a watcher and get started.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/domluna/watcher"
)

func main() {

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	w, err := watcher.NewWatcher(dir)
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})
	defer func() {
		close(done)
	}()

	// Start watching our current directory
	fchan := w.Watch()

	// receive messages from fchan
	go func() {
		for {
			select {
			// case _ = <-fchan:
			case fi, ok := <-fchan:
				// no more messages, exit goroutine and signal done
				if !ok {
					done <- struct{}{}
					return
				}
				fmt.Println("recieved:", fi)
			}
		}
	}()

	go func() {
		// shutdown the watcher
		time.Sleep(3 * time.Second)
		w.Close()
	}()

	<-done
	fmt.Println("WE DONE!")
}
