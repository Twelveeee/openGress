package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestPersistenceManagerEnqueueLatestKeepsNewest(t *testing.T) {
	var mu sync.Mutex
	called := make([]int, 0, 2)
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})

	save := func(snapshot state.Snapshot) error {
		if snapshot.Version == 1 {
			select {
			case <-firstStarted:
			default:
				close(firstStarted)
			}
			<-releaseFirst
		}
		mu.Lock()
		called = append(called, snapshot.Version)
		mu.Unlock()
		return nil
	}

	manager := NewPersistenceManager(save)
	manager.Start()

	manager.EnqueueLatest(state.Snapshot{Version: 1})
	<-firstStarted
	manager.EnqueueLatest(state.Snapshot{Version: 2})
	manager.EnqueueLatest(state.Snapshot{Version: 3})
	close(releaseFirst)

	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := len(called)
		mu.Unlock()
		if n >= 2 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	manager.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(called) != 2 {
		t.Fatalf("expected 2 saves, got %d (%v)", len(called), called)
	}
	if called[0] != 1 {
		t.Fatalf("expected first saved version 1, got %d", called[0])
	}
	if called[1] != 3 {
		t.Fatalf("expected latest saved version 3, got %d", called[1])
	}
}

func TestPersistenceManagerFlushBestEffortSuccess(t *testing.T) {
	var mu sync.Mutex
	called := make([]int, 0, 1)

	save := func(snapshot state.Snapshot) error {
		mu.Lock()
		called = append(called, snapshot.Version)
		mu.Unlock()
		return nil
	}

	manager := NewPersistenceManager(save)
	manager.Start()
	defer manager.Close()

	err := manager.FlushBestEffort(500*time.Millisecond, state.Snapshot{Version: 7})
	if err != nil {
		t.Fatalf("flush expected nil error, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(called) == 0 || called[len(called)-1] != 7 {
		t.Fatalf("expected version 7 to be saved, got %v", called)
	}
}

func TestPersistenceManagerFlushBestEffortTimeout(t *testing.T) {
	save := func(snapshot state.Snapshot) error {
		time.Sleep(120 * time.Millisecond)
		return nil
	}

	manager := NewPersistenceManager(save)
	manager.Start()

	err := manager.FlushBestEffort(20*time.Millisecond, state.Snapshot{Version: 9})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	manager.Close()
}

func TestPersistenceManagerCloseIdempotent(t *testing.T) {
	manager := NewPersistenceManager(func(snapshot state.Snapshot) error { return nil })
	manager.Start()
	manager.Close()
	manager.Close()
}
