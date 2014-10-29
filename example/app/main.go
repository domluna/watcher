package main

import (
	"fmt"
	"log"
	"os"
	// "strings"

	"github.com/domluna/watcher"
)

func main() {
	w, err := watcher.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Watching: ", dir)

	err = w.AddFiles(dir)
	// err = w.AddFiles("$HOME/Desktop/the_stars/simple-todos")
	if err != nil {
		log.Fatal(err)
	}

	// Listen part
	done := make(chan struct{})
	defer func() {
		close(done)
	}()

	fchan, errc := w.Watch(done)

	go func() {
		for {
			select {
			case fi := <-fchan:
				fmt.Println(fi)
			case err = <-errc:
				fmt.Println(err)
			}
		}
	}()

	// time.Sleep(10 * time.Second)
	// done <- struct{}{}
	<-done
}
