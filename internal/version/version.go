package version

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

var (
	// Overridden via -ldflags at build time.
	Version   = "0.0.0-dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Version     string `json:"version"`
	GitCommit   string `json:"git_commit"`
	ShortCommit string `json:"short_commit"`
	Dirty       bool   `json:"dirty"`
	BuildTime   string `json:"build_time"`
	GoVersion   string `json:"go_version"`
}

func Get() Info {
	short := GitCommit
	if len(short) > 7 {
		short = GitCommit[:7]
	}
	return Info{
		Version:     emptyToNA(Version),
		GitCommit:   emptyToNA(GitCommit),
		ShortCommit: emptyToNA(short),
		Dirty:       strings.Contains(Version, "-dirty") || strings.Contains(GitCommit, "dirty"),
		BuildTime:   normalizeTime(BuildTime),
		GoVersion:   runtime.Version(),
	}
}

func (i Info) String() string {
	// Keep it compact; long text can overflow Discord embed limits.
	dirty := ""
	if i.Dirty {
		dirty = " (dirty)"
	}
	return fmt.Sprintf("HKTM %s%s — commit %s — built %s — %s",
		i.Version, dirty, i.ShortCommit, i.BuildTime, i.GoVersion)
}

func emptyToNA(v string) string {
	if strings.TrimSpace(v) == "" || v == "unknown" {
		return "n/a"
	}
	return v
}

func normalizeTime(v string) string {
	// Try to normalize any reasonable build time into RFC3339, else return original/n/a.
	if v == "" || v == "unknown" {
		return "n/a"
	}
	// common formats: RFC3339 already, or "2006-01-02T15:04:05Z"
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	// try a common fallback
	if t, err := time.Parse("2006-01-02T15:04:05Z", v); err == nil {
		return t.UTC().Format(time.RFC3339)
	}
	return v
}
