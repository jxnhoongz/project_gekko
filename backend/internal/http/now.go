package http

import "time"

// timeNow is a small indirection so tests can override the clock.
var timeNow = func() time.Time { return time.Now() }
