package logbuffer

import (
	"fmt"
	"sync"
	"time"
)

type Severity string

const (
	SeverityInfo  Severity = "INFO"
	SeverityWarn  Severity = "WARN"
	SeverityError Severity = "ERROR"
)

type Entry struct {
	Time      time.Time
	Severity  Severity
	Component string
	Message   string
}

type Buffer struct {
	mu      sync.RWMutex
	maxSize int
	entries []Entry
}

func New(maxSize int) *Buffer {
	if maxSize <= 0 {
		maxSize = 500
	}
	return &Buffer{
		maxSize: maxSize,
		entries: make([]Entry, 0, maxSize),
	}
}

func (b *Buffer) Add(severity Severity, component, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.entries = append(b.entries, Entry{
		Time:      time.Now(),
		Severity:  severity,
		Component: component,
		Message:   message,
	})
	if len(b.entries) > b.maxSize {
		b.entries = append([]Entry(nil), b.entries[len(b.entries)-b.maxSize:]...)
	}
}

func (b *Buffer) Addf(severity Severity, component, format string, args ...any) {
	b.Add(severity, component, fmt.Sprintf(format, args...))
}

func (b *Buffer) Snapshot(limit int) []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit >= len(b.entries) {
		return append([]Entry(nil), b.entries...)
	}
	return append([]Entry(nil), b.entries[len(b.entries)-limit:]...)
}

func (b *Buffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = b.entries[:0]
}
