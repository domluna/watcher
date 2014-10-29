package watcher

import "strings"

// Ignorer is a function that takes a string and returns
// true if the string should be ignored, false otherwise.
type Ignorer func(string) bool

// IgnoreFirstDot ignores any file/directory that starts with a "."
func IgnoreDotfiles(s string) bool {
	return strings.HasPrefix(s, ".")
}
