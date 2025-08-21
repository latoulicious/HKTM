package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of HKTM
	Version = "0.1.0"

	// GitCommit is the git commit hash (set during build)
	GitCommit = "unknown"

	// BuildTime is when the binary was built (set during build)
	BuildTime = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("HKTM v%s (commit: %s, built: %s, go: %s)",
		i.Version, i.GitCommit, i.BuildTime, i.GoVersion)
}
