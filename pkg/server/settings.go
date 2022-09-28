package server

import (
	"os"
	"strings"
	"time"
)

var (
	CacheTtl, _ = time.ParseDuration(os.Getenv("CACHE_TTL"))
	LockTtl, _  = time.ParseDuration(os.Getenv("LOCK_TTL"))
	ProjectName = os.Getenv("PROJECT_NAME")

	UseLock               = strings.ToLower(os.Getenv("USE_LOCK")) == "true"
	LockLocation Location = NewLocation(os.Getenv("LOCK_LOCATION"))
	LockIndex    int      = 1                     // only for path location
	LockKey      string   = os.Getenv("LOCK_KEY") // only for query location
)

type Location string

var (
	LocationUnspecified Location = ""
	LocationPath        Location = "path"
	LocationQuery       Location = "query"
)

func NewLocation(s string) Location {
	var l Location
	l.FromString(s)
	return l
}

func (l *Location) FromString(s string) {
	if strings.ToLower(s) == "query" {
		*l = LocationQuery
	} else {
		*l = LocationPath
	}
}
