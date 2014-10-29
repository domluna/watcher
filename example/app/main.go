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
				if !ok {
					done <- struct{}{}
					return
				}
				fmt.Println("RECIEVED:", fi, ok)
			default:
			}
		}
	}()

	w.Close()
	<-done
	time.Sleep(5 * time.Second)
	fmt.Println("WE DONE!")
}
