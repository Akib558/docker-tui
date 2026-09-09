package ui

import "time"

// nowFunc returns the current time. Rendering code MUST use this rather than
// time.Now so that snapshot tests can pin the clock and produce stable output.
var nowFunc = time.Now

// since is nowFunc-aware replacement for time.Since.
func since(t time.Time) time.Duration { return nowFunc().Sub(t) }
