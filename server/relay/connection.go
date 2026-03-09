package relay

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/doraemonkeys/WindSend-Relay/server/protocol"
	"github.com/doraemonkeys/doraemon/crypto"
	"go.uber.org/zap"
)

// Connection represents a single registered Rust device connection.
// In the new multi-connection model, Connection no longer has a Relaying flag
// or a per-connection mutex: "in pool = idle, out of pool = busy/closed".
type Connection struct {
	ID         string
	AuthkeyB64 string
	Cipher     crypto.SymmetricCipher

	Conn        net.Conn
	ConnectTime time.Time
}

// sendMsgDetectAlive sends a heartbeat and waits for a response to probe liveness.
// Must NOT be called while the connection is in the pool (caller must have
// removed it first to avoid concurrent writes on net.Conn).
func (c *Connection) sendMsgDetectAlive() (alive bool) {
	l := zap.L().With(zap.String("id", c.ID), zap.String("addr", c.Conn.RemoteAddr().String()))

	err := protocol.SendHeartbeat(c.Conn, c.ID, c.Cipher)
	if err != nil {
		l.Warn("sent heartbeat failed(detect alive)", zap.Error(err))
		return false
	}

	// Use a read deadline instead of a goroutine+select so we never leak a
	// goroutine blocked on ReadReqHead after the timeout elapses.
	_ = c.Conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	defer func() { _ = c.Conn.SetReadDeadline(time.Time{}) }()

	head, err := protocol.ReadReqHead(c.Conn, c.Cipher)
	if err != nil {
		l.Warn("Failed to receive heartbeat", zap.Error(err))
		return false
	}
	if head.Action != protocol.ActionHeartbeat {
		l.Warn("unexpected action during heartbeat probe", zap.Any("action", head.Action))
		return false
	}
	return true
}

// DeviceConnPool manages a pool of idle connections for a single device ID.
//
// # Connection lifecycle ("use-once, replenish immediately")
//
// Each Rust↔Go TCP connection serves exactly one relay bridge session:
//
//	register → idle in pool → tryAcquire (pops out) → io.Copy bridge → close
//
// After a bridge ends, Go closes the connection. Rust immediately reconnects
// to maintain one idle connection for the next request.
//
// This is the session-level lifecycle. Within a single bridge, the
// Flutter↔Rust TLS stream is long-lived: Rust's main_process handles
// multiple requests (file chunks, clipboard, etc.) on the same TLS
// connection — identical to the direct-connect path.
//
// # Concurrency model
//
// "In pool = idle": connections leave the pool via tryAcquire (for relay)
// or borrow-out (for heartbeat probing). No Relaying flag is needed.
type DeviceConnPool struct {
	mu    sync.Mutex
	conns []*Connection

	// notifyCh is a buffered-1 channel used to wake waiters when a new
	// idle connection becomes available (registration, cascade, probe return).
	notifyCh chan struct{}

	// activeCount tracks connections that have been popped by tryAcquire
	// and are currently in a relay bridge.
	activeCount atomic.Int32

	// pendingCount tracks connections that have passed the capacity check
	// but are not yet activated (OK response not yet sent to the client).
	pendingCount atomic.Int32

	// probingCount tracks connections that have been borrowed out by the
	// heartbeat scanner and are currently being probed.
	probingCount atomic.Int32

	// waiterCount tracks goroutines blocked in waitForConnection, used to
	// prevent pool cleanup from orphaning waiters and to cap goroutine buildup.
	waiterCount atomic.Int32

	// lastRelayTime records the timestamp (UnixMilli) of the most recent
	// activeCount decrement. Used for BUSY vs OFFLINE determination and to
	// delay pool cleanup during the Rust reconnect window.
	lastRelayTime atomic.Int64

	// epoch is incremented by admin close/:id. Heartbeat probe return checks
	// epoch consistency to prevent stale connections from being re-inserted
	// after an admin wipe.
	epoch atomic.Int64
}

func newDeviceConnPool() *DeviceConnPool {
	return &DeviceConnPool{
		conns:    make([]*Connection, 0, 2),
		notifyCh: make(chan struct{}, 1),
	}
}

// tryAcquire pops the head connection from the pool. Returns nil if pool is empty.
// On success, activeCount is incremented. If the pool still has remaining connections
// after the pop, a cascade notification is sent to wake the next waiter.
func (p *DeviceConnPool) tryAcquire() *Connection {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.conns) == 0 {
		return nil
	}
	conn := p.conns[0]
	p.conns[0] = nil // Allow GC of the Connection before backing array is reclaimed.
	p.conns = p.conns[1:]
	p.activeCount.Add(1)
	// Cascade notify: if pool still has idle connections, wake the next waiter.
	// Prevents waiter starvation when multiple connections register rapidly
	// and the buffer-1 channel drops notifications.
	if len(p.conns) > 0 {
		select {
		case p.notifyCh <- struct{}{}:
		default:
		}
	}
	return conn
}

// activate inserts a previously-reserved connection into the idle queue and
// notifies waiters. Must be called only after the client has acknowledged OK,
// so the connection is truly ready for relay.
func (p *DeviceConnPool) activate(c *Connection) {
	p.mu.Lock()
	// Keep the reservation visible until the idle connection is actually present
	// in the pool. Otherwise cleanup can observe "no pending + no idle" and
	// delete a just-created pool while activation is still blocked on p.mu.
	p.pendingCount.Add(-1)
	p.conns = append(p.conns, c)
	p.mu.Unlock()
	select {
	case p.notifyCh <- struct{}{}:
	default:
	}
}

// totalLocked returns the total connection count while the caller already holds p.mu.
// Includes idle, active, probing, and pending (reserved but not yet activated) connections.
func (p *DeviceConnPool) totalLocked() int {
	return len(p.conns) + int(p.activeCount.Load()) + int(p.probingCount.Load()) + int(p.pendingCount.Load())
}
