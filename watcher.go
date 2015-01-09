package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

// Strings prints out the string representation of a FileEvent.
func (fe FileEvent) String() string {
	return fmt.Sprintf("{\n\tPath: %s\n\tName: %s\n\tExt: %s\n\tOp: %s \n}",
		fe.Path, fe.Name, fe.Ext, fe.Op)
}

// Watcher watches files for changes. It recursively
// add files to be watched.
type Watcher struct {
	fsw        *fsnotify.Watcher
	extensions []string
	done       chan struct{}
	closed     bool

	Events chan *FileEvent
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
	w.closed = true
	w.done <- struct{}{}
}

// New creates a Watcher. The Watcher watches files
// recursively from the root.
//
// Any number of extensions may be passed, if extensions are
// passed the watcher only watches files with respect to the
// extensions.
//
// Prefix the extension with a ".", for example go files would be
// passed as ".go".
func New(root string, extensions ...string) (*Watcher, error) {
	w := Watcher{
		done: make(chan struct{}),
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w.fsw = fsw

	w.extensions = extensions
	w.Events = make(chan *FileEvent, 10)

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

	err := w.walkFS(root)
	if err != nil && err != filepath.SkipDir {
		return err
	}
	return nil
}

// Watch watches stuff.
func (w *Watcher) Watch() {
	go func() {
		defer func() {
			close(w.Events)
			w.done <- struct{}{}
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
						w.fsw.Add(pe.Path)
					}
				case Remove:
					w.fsw.Remove(pe.Path)
				}

				w.Events <- pe
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
}

// walkFS walks the filesystem and recursively adds files/directories.
func (w *Watcher) walkFS(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			if shouldIgnore(name) && name != "." && name != ".." {
				return filepath.SkipDir
			}
		}

		// check if file is valid
		if info.Mode().IsRegular() && !w.validFile(path) {
			return nil
		}

		w.fsw.Add(path)
		return nil
	})
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

// validFile determines whether a path should be
// watched.
func (w *Watcher) validFile(path string) bool {
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	return !shouldIgnore(name) && w.keep(ext)
}

// keep determines whether file, descrbed by ext
// should be kept.
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

// shouldIgnore ignores files that are prefixed with a dot
// or underscore.
func shouldIgnore(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}
