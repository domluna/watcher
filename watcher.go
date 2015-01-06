package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-fsnotify/fsnotify"
)

// Op describes a set of file operations. Wraps fsnotify.
type Op uint32

const (
	Create Op = iota << 1
	Write
	Remove
	Rename
	Chmod
)

// FileEvent wraps information about the file events
// from fsnotify.
type FileEvent struct {
	// Absolute path of file.
	Path string
	// Name of the file.
	Name string
	// The file extension, ex. html, js
	Ext string
	// The operation that triggered the event
	Op
}

func (fe FileEvent) String() string {
	return fmt.Sprintf("{\n\tPath: %s\n\tName: %s\n\tExt: %s\n\tOp: %s \n}",
		fe.Path, fe.Name, fe.Ext, fe.Op)
}

// Watcher watches files for changes. It recursively
// add files to be watched.
type Watcher struct {
	fsw *fsnotify.Watcher

	files map[string]struct{}

	extensions []string

	done chan struct{}

	closed bool

	verbose bool
}

// wait waits for the watcher to shut down.
func (w *Watcher) wait() {
	defer func() {
		close(w.done)
	}()
	for {
		select {
		case <-w.done:
			w.fsw.Close()
			return
		default:
		}
	}
}

// Close closes the watcher.
func (w *Watcher) Close() {
	if w.closed {
		return
	}
	log.Println("CLOSING WATCHER")
	w.closed = true
	w.done <- struct{}{}
}

// New creates a Watcher.
func New(root string, types ...string) (*Watcher, error) {
	w := Watcher{
		done: make(chan struct{}),
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w.fsw = fsw

	w.extensions = types

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
func (w *Watcher) Watch(verbose bool) <-chan *FileEvent {
	w.verbose = verbose
	fchan := make(chan *FileEvent, 5)
	go func() {
		defer func() {
			close(fchan)
			w.done <- struct{}{}
			log.Println("EXITING GOROUTINE")
		}()

		for {
			select {
			case ev, ok := <-w.fsw.Events:
				// If the fsnotify event chan is closed
				// there's no reason for this goroutine to
				// keep running.
				if !ok {
					break
				}
				pe := parseEvent(ev)

				// Remove and add files from the watcher
				switch pe.Op {
				case Create:
					fi, err := os.Stat(pe.Path)
					if err != nil {
						continue
					}

					if fi.IsDir() || w.validFile(pe.Path) {
						log.Println("Adding", pe.Name)
						w.fsw.Add(pe.Path)
					}
				case Remove:
					log.Println("Removing", pe.Name)
					w.fsw.Remove(pe.Path)
				}

				fchan <- pe
			case err, ok := <-w.fsw.Errors:
				// If the channel is closed done has
				// already been shutdown.
				if !ok {
					break
				}

				log.Fatal(err)
			}
		}
	}()

	return fchan
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
			if ignore(filepath.Base(info.Name())) && info.IsDir() {
				log.Println("Ignoring:", filepath.Base(info.Name()))
				return filepath.SkipDir
			}

			// if ignore(filepath.Base(info.Name())) {
			// 	log.Println("Ignoring:", filepath.Base(info.Name()))
			// 	return nil
			// }
			//
			// if !w.keep(filepath.Ext(path)) {
			// 	log.Println("Ignoring:", filepath.Base(info.Name()))
			// 	return nil
			// }

			if info.Mode().IsRegular() && !w.validFile(path) {
				log.Println("Ignoring:", filepath.Base(info.Name()))
				return nil
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Println("Initially adding:", info.Name())
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

// parseEvent parses the event wrapping it into a filevent
// making it easier to work with.
func parseEvent(ev fsnotify.Event) *FileEvent {
	spl := strings.Split(ev.String(), ": ")

	fi := &FileEvent{}

	if len(spl) > 0 {
		path := spl[0]

		path = strings.Trim(path, "\"")

		fi.Ext = filepath.Ext(path)
		fi.Name = filepath.Base(path)
		fi.Path = path

		switch ev.Op {
		case fsnotify.Create:
			fi.Op = Create
		case fsnotify.Chmod:
			fi.Op = Chmod
		case fsnotify.Write:
			fi.Op = Write
		case fsnotify.Remove:
			fi.Op = Remove
		case fsnotify.Rename:
			fi.Op = Rename
		}
	}

	return fi
}

func (w *Watcher) validFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	return !ignore(name) && w.keep(ext)
}

func (w *Watcher) keep(ext string) bool {
	// Any file extension is kept
	if len(w.extensions) == 0 {
		return true
	}

	for _, ex := range w.extensions {
		if ex == ext {
			return true
		}
	}
	return false
}

// debug prints the message
func (w *Watcher) debug(msg string) {
	if w.verbose {
		log.Println(msg)
	}
}

// ignore ignores files that are prefixed with a dot
// or underscore.
func ignore(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
