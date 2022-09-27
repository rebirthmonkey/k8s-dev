package version

import "fmt"

var Version string
var Commit string

// Info get version info
func Info() string {
	//return fmt.Sprintf("%s (%s)", Version, Commit)
	return fmt.Sprintf("%s (%s)", "v0.1", "xxxx")
}
