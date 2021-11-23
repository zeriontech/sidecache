package server

import (
	"os"
	"strings"
	"time"
)

var (
	CacheTtl, _ = time.ParseDuration(os.Getenv("CACHE_TTL"))
	LockTtl, _ = time.ParseDuration(os.Getenv("LOCK_TTL"))
	UseLock    = strings.ToLower(os.Getenv("USE_LOCK")) == "true"
)
