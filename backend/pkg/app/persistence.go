package app

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type persistTask struct {
	snapshot state.Snapshot
	done     chan error
}

type snapshotSaver func(state.Snapshot) error

// PersistenceManager 通过单 writer + 有界队列执行异步持久化。
type PersistenceManager struct {
	save   snapshotSaver
	queue  chan persistTask
	stopCh chan struct{}
	doneCh chan struct{}

	mu      sync.RWMutex
	started bool
	stopped bool
}

func NewPersistenceManager(save snapshotSaver) *PersistenceManager {
	return &PersistenceManager{
		save:   save,
		queue:  make(chan persistTask, 1),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (m *PersistenceManager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	go m.loop()
}

func (m *PersistenceManager) loop() {
	defer close(m.doneCh)
	for {
		select {
		case task := <-m.queue:
			err := m.save(task.snapshot)
			if err != nil {
				slog.Error("persistence save failed", "err", err)
			}
			if task.done != nil {
				task.done <- err
				close(task.done)
			}
		case <-m.stopCh:
			return
		}
	}
}

func (m *PersistenceManager) EnqueueLatest(snapshot state.Snapshot) {
	if !m.active() {
		return
	}
	m.enqueueReplace(persistTask{snapshot: snapshot})
}

func (m *PersistenceManager) FlushBestEffort(timeout time.Duration, snapshot state.Snapshot) error {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if !m.active() {
		if m.save == nil {
			return errors.New("persistence saver is nil")
		}
		return m.save(snapshot)
	}

	task := persistTask{
		snapshot: snapshot,
		done:     make(chan error, 1),
	}
	m.enqueueReplace(task)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case err := <-task.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *PersistenceManager) Close() {
	m.mu.Lock()
	if !m.started || m.stopped {
		m.mu.Unlock()
		return
	}
	m.stopped = true
	close(m.stopCh)
	m.mu.Unlock()

	<-m.doneCh
}

func (m *PersistenceManager) enqueueReplace(task persistTask) {
	for {
		select {
		case m.queue <- task:
			return
		default:
			select {
			case <-m.queue:
			default:
			}
		}
	}
}

func (m *PersistenceManager) active() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started && !m.stopped
}
