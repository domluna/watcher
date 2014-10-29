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
	return fmt.Sprintf("%s", fe.Name)
}

// Watcher watches files for changes
type Watcher struct {
	fsw *fsnotify.Watcher

	files map[string]struct{}

	ignorers []Ignorer
	fchan    chan *FileEvent
	done     chan struct{}

	isClosed bool
}

func (w *Watcher) wait() {
	defer func() {
		close(w.done)
	}()
	for {
		select {
		case <-w.done:
			w.fsw.Close()
			close(w.fchan)
			return
		default:
		}
	}
}

// Close the watcher.
func (w *Watcher) Close() {
	if w.isClosed {
		return
	}
	log.Println("CLOSING WATCHER")
	log.Println("SENDING TO DONE")
	w.isClosed = true
	w.done <- struct{}{}
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
func NewWatcher(root string, options ...func(*Watcher) error) (*Watcher, error) {
	w := Watcher{
		done:  make(chan struct{}),
		fchan: make(chan *FileEvent),
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w.fsw = fsw
	w.ignorers = append(w.ignorers, IgnoreDotfiles)

	if err = w.SetOptions(options...); err != nil {
		return nil, err
	}

	err = w.addFiles(root)
	if err != nil {
		return nil, err
	}

	// Wait for the close signal.
	go w.wait()
	return &w, nil
}


// AddFiles starts to recurse from the root and add files to
// the watch list.
func (w *Watcher) addFiles(root string) error {
	root = os.ExpandEnv(root)
	errc := w.walkFS(root)

	if err := <-errc; err != nil && err != filepath.SkipDir {
		return err
	}
	return nil
}

// Watch watches stuff.
func (w *Watcher) Watch() <-chan *FileEvent {
	go func() {
		defer func() {
			log.Println("EXITING GOROUTINE")
		}()

		for {
			select {
			case ev, ok := <-w.fsw.Events:
				log.Println("EVENT", ev, ok)

				// If the fsnotify event chan is closed
				// there's no reason for this goroutine to
				// keep running.
				if !ok {
					log.Println("EXITING WATCH CHAN")
					return
				}

				fi := &FileEvent{
					Name: ev.String(),
					// Op: ev.Op,
				}
				w.fchan <- fi
			case err, ok := <-w.fsw.Errors:
				log.Println("ERROR", err, ok)

				// If the channel is closed done has
				// already been shutdown.
				if !ok {
					log.Println("EXITING WATCH CHAN")
					return
				}
				w.done <- struct{}{}
				return
			}
		}
	}()

	return w.fchan
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

			log.Println(path)
			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Println("ADDING:", info.Name())
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

func parseEvent(ev fsnotify.Event) *FileEvent {
}
