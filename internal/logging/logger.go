// Package logging writes JSON Lines runtime logs. It has no password or DSN fields.
package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cznUSER/B-HIS/internal/redact"
)

// Event is one JSON line. Field names follow 工程约定 §8.3.
type Event struct {
	Timestamp    string `json:"timestamp"`
	Level        string `json:"level"`
	RequestID    string `json:"request_id"`
	ActorID      string `json:"actor_id"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Outcome      string `json:"outcome"`
	ErrorCode    string `json:"error_code"`
	DurationMS   int64  `json:"duration_ms"`
	Message      string `json:"message,omitempty"`
}

// Logger appends to {Dir}/h-api-YYYYMMDD.log (Asia/Shanghai date).
type Logger struct {
	dir  string
	min  int
	loc  *time.Location
	mu   sync.Mutex
	file *os.File
	day  string
}

func New(dir, level string) (*Logger, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	return &Logger{dir: dir, min: rank(level), loc: loc}, nil
}

func (l *Logger) Dir() string { return l.dir }

func (l *Logger) Log(e Event) {
	if rank(e.Level) < l.min {
		return
	}
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	e.Level = strings.ToLower(e.Level)
	line, err := json.Marshal(e)
	if err != nil {
		return
	}
	line = []byte(redact.String(string(line)))
	l.mu.Lock()
	defer l.mu.Unlock()
	day := time.Now().In(l.loc).Format("20060102")
	if l.file == nil || l.day != day {
		if l.file != nil {
			_ = l.file.Close()
		}
		path := filepath.Join(l.dir, "h-api-"+day+".log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
		if err != nil {
			return
		}
		l.file = f
		l.day = day
	}
	_, _ = l.file.Write(append(line, '\n'))
	_ = l.file.Sync()
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func rank(level string) int {
	switch strings.ToLower(level) {
	case "debug":
		return 0
	case "info":
		return 1
	case "warn", "warning":
		return 2
	case "error":
		return 3
	default:
		return 1
	}
}
