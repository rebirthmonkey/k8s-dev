package version

import "fmt"

var Version string
var Commit string

// Info get version info
func Info() string {
	return fmt.Sprintf("%s (%s)", Version, Commit)
}
