package watcher

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var (
	testDir    string
	writeFile1 *os.File
	writeFile2 *os.File
	ignoreFile *os.File
)

const (
	testDuration  = time.Second * 3
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

	writeFile1, err = ioutil.TempFile(t1, "bar1")
	if err != nil {
		panic(err)
	}

	writeFile2, err = ioutil.TempFile(testDir, "bar2")
	if err != nil {
		panic(err)
	}

	ignoreFile, err = ioutil.TempFile(t3, "bazz")
	if err != nil {
		panic(err)
	}
}

// Any changes to files stored in dotfile directories should be ignored
func Test_IgnoreDotfiles(t *testing.T) {
	done := make(chan struct{})
	defer func() {
		close(done)
		ignoreFile.Close()
	}()

	w, err := NewWatcher(testDir)
	if err != nil {
		t.Fatal(err)
	}
	fchan := w.Watch()
	go func() {
		for {
			select {
			// We should never get anything out this channel
			case <-fchan:
				t.Error(err)
			default:
			}

		}
	}()

	// Write to the ignored file
	go func() {
		for {
			select {
			case <-done:
				// w.Close()
				return
			default:
				ignoreFile.Write([]byte("HELLO"))
				time.Sleep(writeInterval)
			}
		}
	}()
	time.Sleep(testDuration)
	done <- struct{}{}
}

// We're writing
func Test_DataRace(t *testing.T) {
	done := make(chan struct{})
	defer func() {
		close(done)
		writeFile1.Close()
		writeFile2.Close()
	}()

	w, err := NewWatcher(testDir)
	if err != nil {
		t.Fatal(err)
	}
	fchan := w.Watch()

	go func() {
		time.Sleep(time.Second * 2)
		w.Close()
	}()

	go func() {
		for {
			select {
			case _, ok := <-fchan:
				if !ok {
					return
				}
			default:
			}

		}
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				writeFile1.Write([]byte("HELLO"))
				writeFile2.Write([]byte("HELLO"))
				time.Sleep(writeInterval)
			}
		}
	}()
	time.Sleep(testDuration)
	done <- struct{}{}
}
