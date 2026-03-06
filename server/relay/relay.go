package relay

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/doraemonkeys/WindSend-Relay/server/config"
	"github.com/doraemonkeys/WindSend-Relay/server/protocol"
	"github.com/doraemonkeys/WindSend-Relay/server/relay/auth"
	"github.com/doraemonkeys/WindSend-Relay/server/storage"
	"github.com/doraemonkeys/WindSend-Relay/server/tool"
	"github.com/doraemonkeys/doraemon"
	"github.com/doraemonkeys/doraemon/crypto"
	"go.uber.org/zap"
)

const (
	// maxConnsPerDevice is the per-ID total connection cap (idle + active + probing).
	maxConnsPerDevice = 16
	// maxWaitersPerDevice caps the number of goroutines that may block waiting
	// for an idle connection before returning DEVICE_BUSY immediately.
	maxWaitersPerDevice int32 = 8
	// waitTimeout is how long a relay request will wait for a new idle connection.
	waitTimeout = 3 * time.Second
	// reconnectWindow is the grace period after the last relay ends, during which
	// we assume Rust is reconnecting. Prevents premature OFFLINE verdicts and
	// premature pool cleanup.
	reconnectWindow = 5 * time.Second
	// denyTTL is how long an admin-denied device ID stays rejected.
	denyTTL = 5 * time.Minute
)

// Sentinel errors for the wait path.
var (
	errDeviceBusy    = errors.New("device busy")
	errDeviceOffline = errors.New("device offline")
)

type SecretLimit struct {
	count atomic.Int32
	limit int
}

type Relay struct {
	config config.Config
	// nil when no secret keys
	authenticator *auth.Authentication
	storage       storage.Storage

	// ID -> DeviceConnPool
	connections   map[string]*DeviceConnPool
	connectionsMu sync.RWMutex

	// globalConnCount tracks the true total number of registered connections
	// across all device IDs (atomic, not derived from map size).
	globalConnCount atomic.Int32

	keyConnLimit   map[string]*SecretLimit
	keyConnLimitMu sync.RWMutex

	// denyList stores device IDs that have been administratively denied.
	// Value is the Unix-milli timestamp when the deny was issued.
	// Protected by denyListMu; independent of DeviceConnPool lifecycle.
	denyList   map[string]int64
	denyListMu sync.RWMutex

	idRateLimiter *doraemon.RateLimiter
	ipRateLimiter *doraemon.RateLimiter
}

func NewRelay(config config.Config, storage storage.Storage) *Relay {

	var rawSecretKeys []string
	for _, secret := range config.SecretInfo {
		rawSecretKeys = append(rawSecretKeys, secret.SecretKey)
	}

	at := auth.NewAuthentication(rawSecretKeys)
	rawKeyToAES192Key := at.GetAllAuthKeys()
	if len(rawSecretKeys) == 0 {
		zap.L().Warn("No secret keys, authentication is disabled")
		at = nil
	}
	connLimit := make(map[string]*SecretLimit, len(rawSecretKeys))
	for _, secret := range config.SecretInfo {
		authKeyB64 := base64.StdEncoding.EncodeToString(rawKeyToAES192Key[secret.SecretKey])
		connLimit[authKeyB64] = &SecretLimit{count: atomic.Int32{}, limit: secret.MaxConn}
	}

	if config.EnableAuth && len(rawSecretKeys) == 0 {
		zap.L().Fatal("Enable authentication but no secret keys")
	}
	return &Relay{
		config:        config,
		authenticator: at,
		storage:       storage,
		keyConnLimit:  connLimit,
		connections:   make(map[string]*DeviceConnPool),
		denyList:      make(map[string]int64),
		idRateLimiter: doraemon.NewRateLimiter(120, time.Minute, 6),
		ipRateLimiter: doraemon.NewRateLimiter(1000, time.Minute, 6),
	}
}

func (r *Relay) Run() {
	zap.L().Info("Relay server start")

	go r.detectConnectionAlive()

	listener, err := net.Listen("tcp", r.config.ListenAddr)
	if err != nil {
		zap.L().Fatal("Failed to listen", zap.Error(err))
	}
	zap.L().Info("Listening on", zap.String("addr", r.config.ListenAddr))
	for {
		conn, err := listener.Accept()
		if err != nil {
			zap.L().Error("Failed to accept", zap.Error(err))
			continue
		}
		zap.L().Info("Accepted connection", zap.String("addr", conn.RemoteAddr().String()))
		go r.mainProcess(conn)
	}
}

// --- Status types for admin API ---

type DevicePoolStatus struct {
	ID            string
	IdleCount     int
	ActiveCount   int
	ProbingCount  int
	LastRelayTime int64
	Denied        bool
}

func (r *Relay) GetAllStatus() []DevicePoolStatus {
	r.connectionsMu.RLock()
	statuses := make([]DevicePoolStatus, 0, len(r.connections))
	for id, pool := range r.connections {
		pool.mu.Lock()
		idle := len(pool.conns)
		pool.mu.Unlock()
		statuses = append(statuses, DevicePoolStatus{
			ID:            id,
			IdleCount:     idle,
			ActiveCount:   int(pool.activeCount.Load()),
			ProbingCount:  int(pool.probingCount.Load()),
			LastRelayTime: pool.lastRelayTime.Load(),
		})
	}
	r.connectionsMu.RUnlock()

	// Annotate denied status from the independent denyList.
	r.denyListMu.RLock()
	for i := range statuses {
		if deniedAt, ok := r.denyList[statuses[i].ID]; ok && time.Since(time.UnixMilli(deniedAt)) < denyTTL {
			statuses[i].Denied = true
		}
	}
	// Also include denied IDs that have no pool (pool was cleaned up but deny persists).
	for id, deniedAt := range r.denyList {
		if time.Since(time.UnixMilli(deniedAt)) >= denyTTL {
			continue
		}
		found := false
		for _, s := range statuses {
			if s.ID == id {
				found = true
				break
			}
		}
		if !found {
			statuses = append(statuses, DevicePoolStatus{
				ID:     id,
				Denied: true,
			})
		}
	}
	r.denyListMu.RUnlock()

	return statuses
}

func (r *Relay) GetConnectionStatus(id string) (DevicePoolStatus, bool) {
	r.connectionsMu.RLock()
	pool, ok := r.connections[id]
	r.connectionsMu.RUnlock()
	if !ok {
		// Check if the ID is at least in the denyList.
		r.denyListMu.RLock()
		deniedAt, denied := r.denyList[id]
		r.denyListMu.RUnlock()
		if denied && time.Since(time.UnixMilli(deniedAt)) < denyTTL {
			return DevicePoolStatus{ID: id, Denied: true}, true
		}
		return DevicePoolStatus{}, false
	}
	pool.mu.Lock()
	idle := len(pool.conns)
	pool.mu.Unlock()
	status := DevicePoolStatus{
		ID:            id,
		IdleCount:     idle,
		ActiveCount:   int(pool.activeCount.Load()),
		ProbingCount:  int(pool.probingCount.Load()),
		LastRelayTime: pool.lastRelayTime.Load(),
	}
	r.denyListMu.RLock()
	if deniedAt, ok := r.denyList[id]; ok && time.Since(time.UnixMilli(deniedAt)) < denyTTL {
		status.Denied = true
	}
	r.denyListMu.RUnlock()
	return status, true
}

func (r *Relay) mainProcess(conn net.Conn) {
	if !r.ipRateLimiter.Allow(conn.RemoteAddr().String()) {
		zap.L().Error("IP rate limit exceeded", zap.String("addr", conn.RemoteAddr().String()))
		_ = conn.Close()
		return
	}

	cipher, authKey, err := protocol.Handshake(conn, r.authenticator, r.config.EnableAuth)
	if err == protocol.ErrEmptyKDFSalt {
		cipher, authKey, err = protocol.Handshake(conn, r.authenticator, r.config.EnableAuth)
	}
	if err != nil {
		zap.L().Info("handshake failed", zap.Error(err))
		_ = conn.Close()
		return
	}
	head, err := protocol.ReadReqHead(conn, cipher)
	if err != nil {
		zap.L().Error("Failed to read common request head", zap.Error(err))
		_ = conn.Close()
		return
	}

	switch head.Action {
	case protocol.ActionConnect:
		r.handleConnect(conn, head, cipher, authKey)
	case protocol.ActionPing:
		r.handlePing(conn, head, cipher)
	case protocol.ActionRelay:
		r.handleRelay(conn, head, cipher)
	default:
		zap.L().Error("Unknown action", zap.Any("action", head.Action))
		_ = protocol.SendRespHeadError(conn, head.Action, "Unknown action")
		_ = conn.Close()
	}
}

// --- Secret limit helpers ---

func (r *Relay) getSecretLimit(authKeyB64 string) *SecretLimit {
	if authKeyB64 == "" {
		return nil
	}
	r.keyConnLimitMu.RLock()
	v, ok := r.keyConnLimit[authKeyB64]
	r.keyConnLimitMu.RUnlock()
	if !ok {
		if r.authenticator != nil {
			keyPreview := authKeyB64
			if len(keyPreview) > 8 {
				keyPreview = keyPreview[:8] + "..."
			}
			zap.L().Error("getSecretLimit: unknown auth key, closing connection",
				zap.String("keyPrefix", keyPreview))
			return nil
		}
		// No-auth mode: lazily create with max limit.
		v = &SecretLimit{count: atomic.Int32{}, limit: math.MaxInt32}
		r.keyConnLimitMu.Lock()
		// Double-check after acquiring write lock.
		if existing, ok := r.keyConnLimit[authKeyB64]; ok {
			v = existing
		} else {
			r.keyConnLimit[authKeyB64] = v
		}
		r.keyConnLimitMu.Unlock()
	}
	return v
}

// --- Connection release helpers ---

// releaseConnection closes a connection and decrements the global and per-secret
// quota counters, pairing with the atomic reservation during registration (1.5).
// The caller must separately decrement activeCount or probingCount.
func (r *Relay) releaseConnection(conn *Connection) {
	_ = conn.Conn.Close()
	r.globalConnCount.Add(-1)
	if sl := r.getSecretLimit(conn.AuthkeyB64); sl != nil {
		sl.count.Add(-1)
	}
}

// releaseActiveConnection releases a connection that was in active relay:
// decrements activeCount, updates lastRelayTime, then closes the connection.
func (r *Relay) releaseActiveConnection(pool *DeviceConnPool, conn *Connection) {
	r.releaseConnection(conn)
	pool.activeCount.Add(-1)
	pool.lastRelayTime.Store(time.Now().UnixMilli())
}

// --- Pool cleanup ---

// tryCleanupPool attempts to remove the pool map entry for the given device ID.
// Deletion requires all five conditions to be satisfied simultaneously, plus
// pointer identity (the pool in the map must be the same object we hold).
func (r *Relay) tryCleanupPool(deviceID string, p *DeviceConnPool) {
	r.connectionsMu.Lock()
	defer r.connectionsMu.Unlock()
	// Pointer identity check: prevent deleting a pool that was replaced by a new one.
	if r.connections[deviceID] != p {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.conns) == 0 && p.activeCount.Load() == 0 &&
		p.pendingCount.Load() == 0 && p.probingCount.Load() == 0 &&
		p.waiterCount.Load() == 0 &&
		time.Since(time.UnixMilli(p.lastRelayTime.Load())) >= reconnectWindow {
		delete(r.connections, deviceID)
	}
}

// --- handleConnect ---

func (r *Relay) handleConnect(conn net.Conn, head protocol.ReqHead, cipher crypto.SymmetricCipher, authKey tool.AES192Key) {
	var success bool
	defer func() {
		if !success {
			_ = conn.Close()
		}
	}()

	req, err := protocol.ReadReq[protocol.ConnectionReq](conn, head.DataLen, cipher)
	if err != nil {
		zap.L().Error("Failed to read connection request", zap.Error(err))
		return
	}
	zap.L().Debug("Connection request", zap.String("secretKey ID", req.SecretKeyID))

	if !r.idRateLimiter.Allow(req.SecretKeyID) {
		zap.L().Error("ID rate limit exceeded", zap.String("secretKey ID", req.SecretKeyID))
		_ = protocol.SendRespHeadError(conn, head.Action, "ID rate limit exceeded", cipher)
		return
	}

	deviceID := req.SecretKeyID
	authKeyB64 := ""
	if authKey != nil {
		authKeyB64 = base64.StdEncoding.EncodeToString(authKey)
	}

	// Step 0: denyList check (independent of pool, checked first)
	r.denyListMu.RLock()
	if deniedAt, ok := r.denyList[deviceID]; ok && time.Since(time.UnixMilli(deniedAt)) < denyTTL {
		r.denyListMu.RUnlock()
		zap.L().Info("Device denied by admin", zap.String("id", deviceID))
		_ = protocol.SendRespHeadError(conn, protocol.ActionConnect, "device denied by admin", cipher)
		return
	}
	r.denyListMu.RUnlock()

	// Step 1: Global quota reservation (atomic increment, rollback on failure)
	if r.globalConnCount.Add(1) > int32(r.config.MaxConn) {
		r.globalConnCount.Add(-1)
		zap.L().Error("Too many connections (global)", zap.String("id", deviceID))
		_ = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
		return
	}

	// Step 2: Per-secret quota reservation
	var secretLimit *SecretLimit
	if authKeyB64 != "" {
		secretLimit = r.getSecretLimit(authKeyB64)
		if secretLimit == nil {
			r.globalConnCount.Add(-1) // rollback step 1
			zap.L().Error("Unknown auth key, rejecting connection", zap.String("id", deviceID))
			_ = protocol.SendRespHeadError(conn, protocol.ActionConnect, "internal error", cipher)
			return
		}
		if secretLimit.count.Add(1) > int32(secretLimit.limit) {
			secretLimit.count.Add(-1)
			r.globalConnCount.Add(-1) // rollback step 1
			zap.L().Error("Too many connections (per-secret)", zap.String("id", deviceID))
			_ = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
			return
		}
	}

	// rollbackQuota undoes steps 1 and 2
	rollbackQuota := func() {
		if secretLimit != nil {
			secretLimit.count.Add(-1)
		}
		r.globalConnCount.Add(-1)
	}

	// Build the Connection object before taking locks.
	c := &Connection{
		ID:          deviceID,
		Conn:        conn,
		ConnectTime: time.Now(),
		AuthkeyB64:  authKeyB64,
		Cipher:      cipher,
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAliveConfig(net.KeepAliveConfig{
			Enable: true, Idle: 30 * time.Second, Interval: 15 * time.Second, Count: 6,
		})
	}

	// Step 3: Reserve a slot in the pool (checks per-ID capacity inside).
	// The connection is NOT inserted into the idle queue yet — only a capacity
	// reservation (pendingCount) is made. This must happen before sending OK
	// so that a capacity failure can still be communicated as an error response.
	pool, regErr := r.registerConnectionPending(deviceID, c)
	if regErr != nil {
		rollbackQuota()
		zap.L().Error("Too many connections (per-ID)", zap.String("id", deviceID))
		_ = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
		return
	}

	// Send OK. On failure, release the reserved slot.
	err = protocol.SendRespHeadOk(conn, protocol.ActionConnect, cipher)
	if err != nil {
		zap.L().Error("Failed to send OK", zap.Error(err), zap.String("id", deviceID))
		pool.pendingCount.Add(-1)
		rollbackQuota()
		return
	}

	// Activate: insert into the idle queue and notify waiters.
	pool.activate(c)

	zap.L().Info("Connection established", zap.String("id", deviceID),
		zap.String("addr", conn.RemoteAddr().String()))
	success = true
}

// registerConnectionPending reserves a slot in the pool for the given device
// without inserting the connection. The caller must call pool.activate(c) after
// the client acknowledges OK, or pool.pendingCount.Add(-1) on failure.
func (r *Relay) registerConnectionPending(deviceID string, c *Connection) (*DeviceConnPool, error) {
	// Fast path: pool already exists, read lock suffices.
	r.connectionsMu.RLock()
	pool := r.connections[deviceID]
	if pool != nil {
		pool.mu.Lock()
		if pool.totalLocked() >= maxConnsPerDevice {
			pool.mu.Unlock()
			r.connectionsMu.RUnlock()
			return nil, errors.New("per-device connection limit reached")
		}
		pool.pendingCount.Add(1)
		pool.mu.Unlock()
		r.connectionsMu.RUnlock()
		return pool, nil
	}
	r.connectionsMu.RUnlock()

	// Slow path: pool doesn't exist, upgrade to write lock to create it.
	r.connectionsMu.Lock()
	pool = r.connections[deviceID]
	if pool == nil {
		pool = newDeviceConnPool()
		r.connections[deviceID] = pool
	}
	pool.mu.Lock()
	if pool.totalLocked() >= maxConnsPerDevice {
		pool.mu.Unlock()
		r.connectionsMu.Unlock()
		return nil, errors.New("per-device connection limit reached")
	}
	pool.pendingCount.Add(1)
	pool.mu.Unlock()
	r.connectionsMu.Unlock()
	return pool, nil
}

// --- handlePing ---

func (r *Relay) handlePing(conn net.Conn, _ protocol.ReqHead, cipher crypto.SymmetricCipher) {
	defer conn.Close()

	l := zap.L().With(zap.String("Action", "Ping"), zap.String("addr", conn.RemoteAddr().String()))
	l.Info("Ping request")
	err := protocol.SendRespHeadOk(conn, protocol.ActionPing, cipher)
	if err != nil {
		l.Error("Failed to send OK", zap.Error(err))
		return
	}
}

// --- handleRelay ---

func (r *Relay) handleRelay(conn net.Conn, head protocol.ReqHead, cipher crypto.SymmetricCipher) {
	defer conn.Close()

	now := time.Now()
	relaySuccess := false
	relayDataLen := int64(0)
	relayOffline := false

	l := zap.L().With(zap.String("Action", "Relay"), zap.String("ReqAddr", conn.RemoteAddr().String()))
	req, err := protocol.ReadReq[protocol.RelayReq](conn, head.DataLen, cipher)
	if err != nil {
		l.Error("Failed to read relay request", zap.Error(err))
		return
	}

	if !r.idRateLimiter.Allow(req.SecretKeyID) {
		zap.L().Error("ID rate limit exceeded", zap.String("id", req.SecretKeyID))
		_ = protocol.SendRespHeadError(conn, head.Action, "ID rate limit exceeded", cipher)
		return
	}

	deviceID := req.SecretKeyID
	l = l.With(zap.String("ID", deviceID))
	l.Info("Relay request")
	defer func() {
		r.storage.AddRelayStatistic(deviceID, relaySuccess, relayOffline, int(time.Since(now).Milliseconds()), relayDataLen)
	}()

	// DenyList check: reject relays to administratively denied devices even if
	// the pool still has leftover connections from before the deny was issued.
	r.denyListMu.RLock()
	if deniedAt, ok := r.denyList[deviceID]; ok && time.Since(time.UnixMilli(deniedAt)) < denyTTL {
		r.denyListMu.RUnlock()
		l.Info("Device denied by admin", zap.String("id", deviceID))
		_ = protocol.SendRespHeadError(conn, protocol.ActionRelay, "device denied by admin", cipher)
		return
	}
	r.denyListMu.RUnlock()

	// Look up the pool.
	r.connectionsMu.RLock()
	pool := r.connections[deviceID]
	r.connectionsMu.RUnlock()
	if pool == nil {
		l.Info("device not online", zap.String("id", deviceID))
		relayOffline = true
		_ = protocol.SendRespHead(conn, protocol.ActionRelay, protocol.StatusDeviceOffline, "device not online", cipher)
		return
	}

	// Try to acquire an idle connection.
	targetConn := pool.tryAcquire()
	if targetConn == nil {
		// Fast offline: if the pool is completely drained (no idle, no active,
		// no probing) and the reconnect window has elapsed, skip the wait path
		// to avoid a needless waitTimeout delay.
		if pool.activeCount.Load() == 0 && pool.probingCount.Load() == 0 {
			lrt := pool.lastRelayTime.Load()
			if lrt == 0 || time.Since(time.UnixMilli(lrt)) >= reconnectWindow {
				relayOffline = true
				l.Info("device offline (stale empty pool)", zap.String("id", deviceID))
				_ = protocol.SendRespHead(conn, protocol.ActionRelay, protocol.StatusDeviceOffline, "device not online", cipher)
				r.tryCleanupPool(deviceID, pool)
				return
			}
		}

		// Enter wait path.
		var waitErr error
		targetConn, waitErr = r.waitForConnection(pool)
		if waitErr != nil {
			if errors.Is(waitErr, errDeviceOffline) {
				relayOffline = true
				l.Info("device offline (wait timeout)", zap.String("id", deviceID))
				_ = protocol.SendRespHead(conn, protocol.ActionRelay, protocol.StatusDeviceOffline, "device not online", cipher)
			} else {
				l.Info("device busy (wait timeout)", zap.String("id", deviceID))
				_ = protocol.SendRespHead(conn, protocol.ActionRelay, protocol.StatusDeviceBusy, "device busy", cipher)
			}
			return
		}
	}

	// targetConn acquired (activeCount already incremented by tryAcquire).
	// Register deferred cleanup: close connection + activeCount -1 + pool cleanup.
	defer func() {
		r.releaseActiveConnection(pool, targetConn)
		r.tryCleanupPool(deviceID, pool)
	}()

	// Send success to Flutter before bridging.
	err = protocol.SendRespHeadOKWithMsg(conn, protocol.ActionRelay, "Relay start", cipher)
	if err != nil {
		l.Error("Failed to reply to client relay start", zap.Error(err))
		return
	}

	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetKeepAliveConfig(net.KeepAliveConfig{
			Enable: true, Idle: 2 * time.Second, Interval: 1 * time.Second, Count: 3,
		})
	}

	// Send relay-start command to Rust.
	// Set a short write deadline so silently-dead connections fail fast
	// instead of hanging until the OS TCP timeout (~2 min).
	_ = targetConn.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err = protocol.SendRelayStart(targetConn.Conn, targetConn.Cipher)
	_ = targetConn.Conn.SetWriteDeadline(time.Time{}) // clear deadline
	if err != nil {
		l.Error("Failed to send relay start to targetConn", zap.Error(err))
		return
	}

	// Bridge Flutter <-> Rust.
	err = r.relay(targetConn, conn, &relayDataLen)
	if err != nil {
		l.Error("relay data failed", zap.Error(err))
		return
	}
	zap.L().Debug("relay data success", zap.String("targetConn", targetConn.ID),
		zap.String("reqConn", conn.RemoteAddr().String()))
	relaySuccess = true
}

// waitForConnection blocks until an idle connection is available or timeout.
// Returns errDeviceBusy or errDeviceOffline on timeout depending on pool state.
func (r *Relay) waitForConnection(pool *DeviceConnPool) (*Connection, error) {
	if pool.waiterCount.Add(1) > maxWaitersPerDevice {
		pool.waiterCount.Add(-1)
		return nil, errDeviceBusy
	}
	defer pool.waiterCount.Add(-1)

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	for {
		select {
		case <-pool.notifyCh:
			if conn := pool.tryAcquire(); conn != nil {
				return conn, nil
			}
			// Another goroutine grabbed it; loop back and wait again.
		case <-timer.C:
			// Determine BUSY vs OFFLINE based on pool state.
			active := pool.activeCount.Load()
			probing := pool.probingCount.Load()
			pending := pool.pendingCount.Load()
			if active > 0 || probing > 0 || pending > 0 {
				return nil, errDeviceBusy
			}
			// No active or probing connections. Check lastRelayTime for reconnect window.
			lrt := pool.lastRelayTime.Load()
			if lrt > 0 && time.Since(time.UnixMilli(lrt)) < reconnectWindow {
				return nil, errDeviceBusy
			}
			return nil, errDeviceOffline
		}
	}
}

func (r *Relay) relay(targetConn *Connection, reqConn net.Conn, relayDataLen *int64) error {
	var errCH = make(chan error, 2)
	var activelyTimeOut atomic.Bool
	go func() {
		n, err := io.Copy(targetConn.Conn, reqConn)
		atomic.AddInt64(relayDataLen, int64(n))
		activelyTimeOut.Store(true)
		// Set a deadline in the past to unblock the reverse io.Copy immediately.
		// Expected to fail when targetConn was already closed by the remote side.
		if setErr := targetConn.Conn.SetReadDeadline(time.Now().Add(-time.Second)); setErr != nil {
			zap.L().Debug("set targetConn read deadline (expected if dst closed first)", zap.Error(setErr))
		}
		if err != nil {
			errCH <- fmt.Errorf("reqConn -> targetConn: %w", err)
			return
		}
		errCH <- nil
		zap.L().Debug("reqConn -> targetConn success")
	}()
	go func() {
		n, err := io.Copy(reqConn, targetConn.Conn)
		atomic.AddInt64(relayDataLen, int64(n))
		if !activelyTimeOut.Load() {
			if err != nil {
				errCH <- fmt.Errorf("targetConn -> reqConn: %w", err)
			} else {
				// Clean EOF: the destination (Rust) finished processing and
				// closed the bridge connection — normal in the use-once
				// connection model. The requesting side (Flutter) will close
				// shortly after (~20 ms drain), at which point the forward
				// goroutine also completes and relay() returns nil.
				zap.L().Debug("relay dst finished first (clean EOF)",
					zap.String("reqConn", reqConn.RemoteAddr().String()))
				errCH <- nil
			}
			return
		}
		errCH <- nil
		zap.L().Debug("targetConn -> reqConn success")
	}()
	zap.L().Debug("relay start", zap.String("targetConn", targetConn.Conn.RemoteAddr().String()),
		zap.String("reqConn", reqConn.RemoteAddr().String()))
	var relayErr error
	for range 2 {
		relayErr = <-errCH
		if relayErr != nil {
			break
		}
	}
	return relayErr
}

// --- Admin helpers ---

// CloseDevice is called by the admin API to deny a device and close all idle connections.
func (r *Relay) CloseDevice(id string) {
	// Step 1: Write denyList.
	r.denyListMu.Lock()
	r.denyList[id] = time.Now().UnixMilli()
	r.denyListMu.Unlock()

	// Step 2: Increment epoch + clear pool.
	r.connectionsMu.RLock()
	pool := r.connections[id]
	r.connectionsMu.RUnlock()
	if pool == nil {
		return
	}

	pool.mu.Lock()
	pool.epoch.Add(1)
	idleConns := pool.conns
	pool.conns = nil
	pool.mu.Unlock()

	// Close all idle connections outside the lock.
	for _, c := range idleConns {
		r.releaseConnection(c)
	}

	// Try to clean up the pool entry.
	r.tryCleanupPool(id, pool)
}

// AllowDevice removes a device ID from the denyList (admin manual override).
func (r *Relay) AllowDevice(id string) {
	r.denyListMu.Lock()
	delete(r.denyList, id)
	r.denyListMu.Unlock()
}
