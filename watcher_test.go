package watcher

import (
	"testing"

	"gopkg.in/fsnotify.v1"
)

type MockWatcher struct{}

func (l *MockWatcher) Watch(w *fsnotify.Watcher) {
	done := make(chan struct{})
	go func() {
	}()
}

func TestWatcher(t *testing.T) {

}
