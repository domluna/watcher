package watcher_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/domluna/watcher"
)

var (
	testDir    string
	mdFile     *os.File
	jsFile     *os.File
	ignoreFile *os.File
)

const (
	testDuration  = time.Second * 1
	writeInterval = time.Millisecond * 10
)

func init() {
	var err error
	testDir, err = ioutil.TempDir("", "foo")
	if err != nil {
		panic(err)
	}

	t1, err := ioutil.TempDir(testDir, "t1")
	if err != nil {
		panic(err)
	}

	t3, err := ioutil.TempDir(t1, ".t3")
	if err != nil {
		panic(err)
	}

	mdFile, err = ioutil.TempFile(t1, "bar1.md")
	if err != nil {
		panic(err)
	}

	jsFile, err = ioutil.TempFile(testDir, "bar2.js")
	if err != nil {
		panic(err)
	}

	ignoreFile, err = ioutil.TempFile(t3, "bazz")
	if err != nil {
		panic(err)
	}
}

// Any changes to files stored in dotfile directories should be ignored
func Test_Watcher(t *testing.T) {
	done := make(chan struct{})
	w, err := watcher.New(testDir, []string{"js"})
	if err != nil {
		t.Fatal(err)
	}

	// Write to the ignored file
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				ignoreFile.Write([]byte("HELLO"))
				mdFile.Write([]byte("HELLO"))
				jsFile.Write([]byte("HELLO"))
				time.Sleep(writeInterval)
			}
		}
	}()

	go func() {
		select {
		case fe, ok := <-w.Events:
			if ok {
				if fe.Ext != ".js" {
					t.Fatal("should only be responding to files with .js extension")
				}
			}
		}
	}()

	time.Sleep(testDuration)
	done <- struct{}{}
}
