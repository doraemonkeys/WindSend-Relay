package relay

import (
	"testing"
	"time"
)

func TestActivateKeepsPendingReservationUntilPoolInsert(t *testing.T) {
	pool := newDeviceConnPool()
	pool.pendingCount.Store(1)

	pool.mu.Lock()
	started := make(chan struct{})
	done := make(chan struct{})
	conn := &Connection{ID: "device-a"}

	go func() {
		close(started)
		pool.activate(conn)
		close(done)
	}()

	<-started

	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := pool.pendingCount.Load(); got != 1 {
			pool.mu.Unlock()
			t.Fatalf("activate cleared pending reservation before pool insert: got %d", got)
		}
		select {
		case <-done:
			pool.mu.Unlock()
			t.Fatal("activate returned while the pool lock was still held")
		default:
		}
		time.Sleep(time.Millisecond)
	}

	pool.mu.Unlock()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("activate did not finish after releasing the pool lock")
	}

	if got := pool.pendingCount.Load(); got != 0 {
		t.Fatalf("activate did not clear pending reservation after pool insert: got %d", got)
	}
	if len(pool.conns) != 1 || pool.conns[0] != conn {
		t.Fatal("activate did not insert the connection into the idle pool")
	}
}
