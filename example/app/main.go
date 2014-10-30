package main

import (
	"fmt"
	"log"
	"os"
	"time"
	// "strings"

	"github.com/domluna/watcher"
)

func main() {

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	w, err := watcher.NewWatcher(dir)
	// w, err := watcher.NewWatcher("$HOME/Desktop/the_stars/simple-todos")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Watching: ", dir)

	// Listen part
	done := make(chan struct{})
	defer func() {
		close(done)
	}()

	fchan := w.Watch()

	go func() {
		for {
			select {
			// case _ = <-fchan:
			case fi, ok := <-fchan:
				fmt.Println("RECIEVED:", fi, ok)
				if !ok {
					done <- struct{}{}
					return
				}
			default:
			}
		}
	}()

	time.Sleep(5 * time.Second)
	w.Close()
	<-done
	fmt.Println("WE DONE!")
}
