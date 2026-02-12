package state

import (
	"sync"
	"time"
)

const DefaultLogCapacity = 200

// LogEntry 为最小可用的全局日志结构。
type LogEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	PlayerID  string    `json:"playerId,omitempty"`
	PortalID  string    `json:"portalId,omitempty"`
	Faction   string    `json:"faction,omitempty"`
	MU        int       `json:"mu,omitempty"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// LogStore 为内存日志队列。
type LogStore struct {
	mu       sync.Mutex
	capacity int
	entries  []LogEntry
}

func NewLogStore(capacity int) *LogStore {
	if capacity <= 0 {
		capacity = DefaultLogCapacity
	}
	return &LogStore{
		capacity: capacity,
		entries:  make([]LogEntry, 0, capacity),
	}
}

func (l *LogStore) Add(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	l.entries = append(l.entries, entry)
	if len(l.entries) > l.capacity {
		excess := len(l.entries) - l.capacity
		l.entries = l.entries[excess:]
	}
}

func (l *LogStore) List() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]LogEntry, len(l.entries))
	copy(out, l.entries)
	return out
}
