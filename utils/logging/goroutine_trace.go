/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2024 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 * Purpose: goroutine-local trace ID storage for zero-intrusion log annotation
**/

package logging

import (
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var goroutineTraces sync.Map // int64 → string

// NewTraceID generates a short trace ID: "TRC-" + 8 random hex chars.
func NewTraceID() string {
	return "TRC-" + uuid.New().String()[:8]
}

// SetGoroutineTrace associates traceID with the current goroutine.
// Call from request middleware; pair with defer ClearGoroutineTrace().
func SetGoroutineTrace(id string) {
	goroutineTraces.Store(goid(), id)
}

// ClearGoroutineTrace removes the trace ID for the current goroutine.
func ClearGoroutineTrace() {
	goroutineTraces.Delete(goid())
}

// CurrentTraceID returns the trace ID for the current goroutine, or "".
func CurrentTraceID() string {
	if v, ok := goroutineTraces.Load(goid()); ok {
		return v.(string)
	}
	return ""
}

func goid() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	s := strings.TrimPrefix(string(buf[:n]), "goroutine ")
	id, _ := strconv.ParseInt(strings.Fields(s)[0], 10, 64)
	return id
}
