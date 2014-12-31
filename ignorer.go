package watcher

import "strings"

// IgnoreDotfiles ignores any file/directory that starts with a "."
func IgnoreDotfiles(s string) bool {
	return strings.HasPrefix(s, ".")
}
