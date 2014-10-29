package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/fsnotify.v1"
)

// Op describes a set of file operations. Wraps fsnotify.
type Op fsnotify.Op

// FileEvent wraps information about the file events
// from fsnotify.
type FileEvent struct {
	// Absolute path of file.
	Path string
	// Name of the file.
	Name string
	// The file extension, ex. html,js
	Ext string
	// The operation that triggered the event
	Op
	time.Time
}

// String returns a string representation of a FileEvent.
func (fe FileEvent) String() string {
	// return fmt.Sprintf("FILEVENT:")
	return fmt.Sprintf("%s", fe.Name)
}

// Watcher watches files for changes
type Watcher struct {
	fsw *fsnotify.Watcher

	files map[string]struct{}

	ignorers []Ignorer
}

// Kill shutdowns all currently active channels the Watcher is
// interacting with.
func (w *Watcher) Kill() error {
	log.Println("Attempting to shutdown watcher ...")
	if err := w.fsw.Close(); err != nil {
		return err
	}
	return nil
}

// SetOptions sets options.
func (w *Watcher) SetOptions(options ...func(*Watcher) error) error {
	for _, opt := range options {
		if err := opt(w); err != nil {
			return err
		}
	}
	return nil
}

// NewWatcher creates a Watcher.
func NewWatcher(options ...func(*Watcher) error) (*Watcher, error) {
	w := Watcher{}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w.fsw = fsw
	w.ignorers = append(w.ignorers, IgnoreDotfiles)
	if err = w.SetOptions(options...); err != nil {
		return nil, err
	}
	return &w, nil
}

// AddFiles starts to recurse from the root and add files to
// the watch list.
func (w *Watcher) AddFiles(root string) error {
	root = os.ExpandEnv(root)
	errc := w.walkFS(root)

	if err := <-errc; err != nil && err != filepath.SkipDir {
		return err
	}
	return nil
}

// Watch watches stuff.
func (w *Watcher) Watch(kill chan struct{}) (chan FileEvent, <-chan error) {
	// Size of chan is the number of ops.
	fchan := make(chan FileEvent, 5)
	errc := make(chan error, 1)

	// Start to receive the kill signal.
	go func() {
		for {
			select {
			case _ = <-kill:
				w.Kill()
				return
			}
		}
	}()

	// Start listening on the fsnotify watcher channels
	go func() {
		for {
			select {
			case ev := <-w.fsw.Events:
				log.Println(ev)
				fi := FileEvent{
					Name: ev.String(),
					// Op: ev.Op,
				}
				fchan <- fi
			case err := <-w.fsw.Errors:
				errc <- err
			}
		}
	}()

	return fchan, errc
}

// ignore loops through our ignorers to see if we should ignore
// the path.
func (w *Watcher) ignore(path string) bool {
	for _, i := range w.ignorers {
		if i(path) {
			return true
		}
	}
	return false
}

// walkFS walks the filesystem.
func (w *Watcher) walkFS(root string) <-chan error {
	errc := make(chan error, 1)
	go func() {
		var wg sync.WaitGroup
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// If it's an directory and it matches our ignore
			// clause, then skip looking at the whole directory
			if w.ignore(info.Name()) && info.IsDir() {
				return filepath.SkipDir
			}

			// If the file isn't regular and not a directory, move on.
			if !info.Mode().IsRegular() && !info.IsDir() {
				return nil
			}

			// If a file matches a ignore clause, move on.
			if w.ignore(info.Name()) {
				return nil
			}

			fmt.Println(path)
			wg.Add(1)
			go func() {
				defer wg.Done()
				// check ignore files here
				// fmt.Println(path)
				fmt.Println("ADDING:", info.Name())
				w.fsw.Add(path)
			}()
			return nil
		})

		go func() {
			wg.Wait()
		}()
		errc <- err

	}()
	return errc
}
