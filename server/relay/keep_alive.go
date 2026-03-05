package relay

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

func (r *Relay) detectConnectionAlive() {
	const detectInterval = time.Second * 60
	for {
		time.Sleep(detectInterval)

		// Snapshot all pool pointers + device IDs under the read lock.
		r.connectionsMu.RLock()
		type poolEntry struct {
			id   string
			pool *DeviceConnPool
		}
		entries := make([]poolEntry, 0, len(r.connections))
		for id, pool := range r.connections {
			entries = append(entries, poolEntry{id: id, pool: pool})
		}
		r.connectionsMu.RUnlock()

		for _, entry := range entries {
			r.probePool(entry.id, entry.pool)
		}

		// denyList TTL cleanup at the end of each scan cycle.
		r.denyListMu.Lock()
		for id, deniedAt := range r.denyList {
			if time.Since(time.UnixMilli(deniedAt)) >= denyTTL {
				delete(r.denyList, id)
			}
		}
		r.denyListMu.Unlock()
	}
}

// probePool uses the "borrow-out" model to probe all idle connections in a pool.
// Lock-remove conns + probingCount increment + epoch snapshot, unlock, probe,
// lock-return alive if epoch matches, release dead.
func (r *Relay) probePool(deviceID string, pool *DeviceConnPool) {
	// Borrow out all idle connections.
	pool.mu.Lock()
	if len(pool.conns) == 0 {
		pool.mu.Unlock()
		// No idle connections to probe, but the pool may be stale
		// (all connections died between keepalive cycles). Attempt cleanup
		// so that it doesn't linger in the connections map indefinitely.
		r.tryCleanupPool(deviceID, pool)
		return
	}
	probing := pool.conns
	pool.conns = nil // Must be nil, not [:0], to avoid aliasing the backing array.
	pool.probingCount.Add(int32(len(probing)))
	epochSnapshot := pool.epoch.Load()
	pool.mu.Unlock()

	// Probe all connections concurrently to avoid serial 2-second timeouts
	// compounding into O(N * timeout) worst-case latency.
	type probeResult struct {
		conn  *Connection
		alive bool
	}
	results := make([]probeResult, len(probing))
	var wg sync.WaitGroup
	for i, c := range probing {
		wg.Add(1)
		go func(idx int, conn *Connection) {
			defer wg.Done()
			results[idx] = probeResult{conn: conn, alive: conn.sendMsgDetectAlive()}
		}(i, c)
	}
	wg.Wait()

	var alive, dead []*Connection
	for _, res := range results {
		if res.alive {
			alive = append(alive, res.conn)
		} else {
			dead = append(dead, res.conn)
		}
	}

	// Return alive connections to the pool (if epoch hasn't changed).
	pool.mu.Lock()
	pool.probingCount.Add(-int32(len(probing)))
	if pool.epoch.Load() == epochSnapshot {
		// Epoch unchanged — safe to return alive connections.
		pool.conns = append(pool.conns, alive...)
	} else {
		// Epoch changed (admin close/:id during probing) — don't re-insert.
		dead = append(dead, alive...)
		alive = nil
	}
	pool.mu.Unlock()

	// Notify waiter if we returned any alive connections.
	if len(alive) > 0 {
		select {
		case pool.notifyCh <- struct{}{}:
		default:
		}
	}

	// Close dead connections (decrement global/per-secret counters).
	for _, c := range dead {
		r.releaseConnection(c)
		zap.L().Info("heartbeat probe: connection dead",
			zap.String("id", c.ID), zap.String("addr", c.Conn.RemoteAddr().String()))
	}

	// Trigger pool cleanup after probing (in case all connections died).
	r.tryCleanupPool(deviceID, pool)
}
