package relay

import (
	"encoding/base64"
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

type Relay struct {
	config config.Config
	// nil when no secret keys
	authenticator *auth.Authentication
	storage       storage.Storage
	// ID -> Connection
	connections   map[string]*Connection
	connectionsMu sync.RWMutex
	keyConnLimit  map[string]*struct {
		count atomic.Int32
		limit int
	}
	keyConnLimitMu sync.RWMutex
	idRateLimiter  *doraemon.RateLimiter
	ipRateLimiter  *doraemon.RateLimiter
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
	connLimit := make(map[string]*struct {
		count atomic.Int32
		limit int
	}, len(rawSecretKeys))
	for _, secret := range config.SecretInfo {
		authKeyB64 := base64.StdEncoding.EncodeToString(rawKeyToAES192Key[secret.SecretKey])
		connLimit[authKeyB64] = &struct {
			count atomic.Int32
			limit int
		}{count: atomic.Int32{}, limit: secret.MaxConn}
	}

	if config.EnableAuth && len(rawSecretKeys) == 0 {
		zap.L().Fatal("Enable authentication but no secret keys")
	}
	return &Relay{
		config:        config,
		authenticator: at,
		storage:       storage,
		keyConnLimit:  connLimit,
		connections:   make(map[string]*Connection),
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
		}
		zap.L().Info("Accepted connection", zap.String("addr", conn.RemoteAddr().String()))
		go r.mainProcess(conn)
	}
}

type ConnectionStatus struct {
	ID          string
	ReqAddr     string
	ConnectTime time.Time
	LastActive  time.Time
	Relaying    bool
}

func (r *Relay) GetAllStatus() []ConnectionStatus {
	r.connectionsMu.RLock()
	defer r.connectionsMu.RUnlock()
	statuses := make([]ConnectionStatus, 0, len(r.connections))
	for _, c := range r.connections {
		statuses = append(statuses, ConnectionStatus{
			ID:          c.ID,
			ReqAddr:     c.Conn.RemoteAddr().String(),
			ConnectTime: c.ConnectTime,
			LastActive:  c.LastNormalActive,
			Relaying:    c.Relaying,
		})
	}
	return statuses
}

func (r *Relay) GetConnectionStatus(id string) (ConnectionStatus, bool) {
	r.connectionsMu.RLock()
	defer r.connectionsMu.RUnlock()
	if c, ok := r.connections[id]; ok {
		return ConnectionStatus{
			ID:          c.ID,
			ReqAddr:     c.Conn.RemoteAddr().String(),
			ConnectTime: c.ConnectTime,
			LastActive:  c.LastNormalActive,
			Relaying:    c.Relaying,
		}, true
	}
	return ConnectionStatus{}, false
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

func (r *Relay) checkConnLimitOk(authKeyB64 string) bool {
	r.keyConnLimitMu.RLock()
	v, ok := r.keyConnLimit[authKeyB64]
	r.keyConnLimitMu.RUnlock()
	if !ok {
		if r.authenticator != nil {
			panic("unknown key: " + authKeyB64)
		}
		return true
	}
	return v.count.Load() < int32(v.limit)
}

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
	zap.L().Debug("Connection request", zap.String("id", req.ID))

	if !r.idRateLimiter.Allow(req.ID) {
		zap.L().Error("ID rate limit exceeded", zap.String("id", req.ID))
		_ = protocol.SendRespHeadError(conn, head.Action, "ID rate limit exceeded", cipher)
		return
	}

	if authKey != nil {
		authKeyB64 := base64.StdEncoding.EncodeToString(authKey)
		if !r.checkConnLimitOk(authKeyB64) {
			zap.L().Error("Too many connections", zap.String("id", req.ID))
			err = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
			if err != nil {
				zap.L().Error("Failed to send error", zap.Error(err))
			}
			return
		}
	}

	r.connectionsMu.RLock()
	{
		if c, ok := r.connections[req.ID]; ok {
			r.connectionsMu.RUnlock()
			if c.Relaying {
				zap.L().Warn("connection relay flag is true", zap.String("id", req.ID), zap.String("addr", conn.RemoteAddr().String()))
			}
			if c.SendMsgDetectAlive() {
				zap.L().Error("Connection already exists", zap.String("id", req.ID))
				err = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Connection already exists", cipher)
				if err != nil {
					zap.L().Error("Failed to send error", zap.Error(err))
				}
				return
			}
			zap.L().Info("Existing connection not active, removing it", zap.String("id", req.ID))
			r.RemoveLongConnection(req.ID)

			r.connectionsMu.RLock()
		}
		if len(r.connections) >= r.config.MaxConn {
			r.connectionsMu.RUnlock()

			zap.L().Error("Too many connections", zap.String("id", req.ID))
			err = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
			if err != nil {
				zap.L().Error("Failed to send error", zap.Error(err))
			}
			return
		}
	}
	r.connectionsMu.RUnlock()

	if c, ok := conn.(*net.TCPConn); ok {
		err = c.SetKeepAlive(true)
		if err != nil {
			zap.L().Error("Failed to set keep alive", zap.Error(err))
		}
		err = c.SetKeepAlivePeriod(time.Second * 30)
		if err != nil {
			zap.L().Error("Failed to set keep alive period", zap.Error(err))
		}
	}

	r.AddConnection(req.ID, conn, authKey, cipher)

	err = protocol.SendRespHeadOk(conn, protocol.ActionConnect, cipher)
	if err != nil {
		zap.L().Error("Failed to send OK", zap.Error(err), zap.String("id", req.ID),
			zap.String("addr", conn.RemoteAddr().String()))
		r.RemoveLongConnection(req.ID)
		return
	}

	zap.L().Info("Connection established", zap.String("id", req.ID),
		zap.String("addr", conn.RemoteAddr().String()))
	success = true
}

func (r *Relay) AddConnection(id string, conn net.Conn, authKey tool.AES192Key, cipher crypto.SymmetricCipher) {
	r.connectionsMu.Lock()
	c := &Connection{
		ID:               id,
		Conn:             conn,
		LastNormalActive: time.Now(),
		ConnectTime:      time.Now(),
		Relaying:         false,
		AuthkeyB64:       base64.StdEncoding.EncodeToString(authKey),
		Cipher:           cipher,
	}
	r.connections[id] = c
	r.connectionsMu.Unlock()
	r.addKeyConnCount(c.AuthkeyB64, 1)
}

func (r *Relay) RemoveLongConnection(id string) {
	zap.L().Debug("Remove long connection", zap.String("id", id))
	r.connectionsMu.Lock()
	c := r.removeConnection(id)
	r.connectionsMu.Unlock()
	if c != nil {
		r.addKeyConnCount(c.AuthkeyB64, -1)
	}
}

func (r *Relay) removeConnection(id string) *Connection {
	if c, ok := r.connections[id]; ok {
		_ = c.Conn.Close()
		delete(r.connections, id)
		return c
	}
	return nil
}

func (r *Relay) addKeyConnCount(key string, add int32) (new int32) {
	if key == "" {
		//no auth
		return 0
	}
	r.keyConnLimitMu.RLock()
	v, ok := r.keyConnLimit[key]
	r.keyConnLimitMu.RUnlock()
	if !ok {
		if r.authenticator != nil {
			panic("unknown key: " + key)
		}
		v = &struct {
			count atomic.Int32
			limit int
		}{count: atomic.Int32{}, limit: math.MaxInt32}
		r.keyConnLimitMu.Lock()
		r.keyConnLimit[key] = v
		r.keyConnLimitMu.Unlock()
	}
	return v.count.Add(add)
}

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

	if !r.idRateLimiter.Allow(req.ID) {
		zap.L().Error("ID rate limit exceeded", zap.String("id", req.ID))
		_ = protocol.SendRespHeadError(conn, head.Action, "ID rate limit exceeded", cipher)
		return
	}

	l = l.With(zap.String("ID", req.ID))
	l.Info("Relay request")
	defer func() {
		r.storage.AddRelayStatistic(req.ID, relaySuccess, relayOffline, int(time.Since(now).Milliseconds()), relayDataLen)
	}()

	r.connectionsMu.RLock()
	targetConn, ok := r.connections[req.ID]
	r.connectionsMu.RUnlock()
	if !ok {
		l.Info("device not online", zap.String("id", req.ID))
		relayOffline = true
		err := protocol.SendRespHeadError(conn, protocol.ActionRelay, "device not online", cipher)
		if err != nil {
			l.Error("Failed to send error", zap.Error(err))
		}
		return
	}
	// Simple processing without lock, if targetConn is relaying, return an error
	if targetConn.Relaying {
		// Handle the case where the two requests are too close together, and the previous connection is about to exit
		const maxRetry = 5
		for range maxRetry {
			if !targetConn.Relaying {
				break
			}
			time.Sleep(time.Millisecond * 200)
		}
		if targetConn.Relaying {
			l.Error("Connection is already relaying")
			err := protocol.SendRespHeadError(conn, protocol.ActionRelay, "Connection is already relaying", cipher)
			if err != nil {
				l.Error("Failed to send error", zap.Error(err))
			}
			return
		} else {
			l.Debug("Edge case: previous connection is about to exit, retry")
		}
	}

	err = protocol.SendRespHeadOKWithMsg(conn, protocol.ActionRelay, "Relay start", cipher)
	if err != nil {
		l.Error("Failed to reply to client relay start", zap.Error(err))
		return
	}

	targetConn.Mu.Lock()
	defer func() {
		targetConn.Mu.Unlock()
		targetConn.Relaying = false
	}()
	targetConn.Relaying = true
	defer func() {
		if !relaySuccess {
			r.RemoveLongConnection(targetConn.ID)
			return
		}
		// run in a new goroutine to avoid deadlocks
		go func() {
			// zap.L().Debug("try to read relay end flag")
			alive := targetConn.SendMsgDetectAlive()
			if alive {
				zap.L().Debug("read relay end flag success")
			} else {
				zap.L().Error("targetConn is not alive after relay", zap.String("id", targetConn.ID),
					zap.String("addr", targetConn.Conn.RemoteAddr().String()))
				r.RemoveLongConnection(targetConn.ID)
			}
		}()
	}()
	err = protocol.SendRelayStart(targetConn.Conn, targetConn.Cipher)
	if err != nil {
		l.Error("Failed to send relay start to targetConn", zap.Error(err))
		return
	}
	err = r.relay(targetConn, conn, &relayDataLen)
	if err != nil {
		l.Error("relay data failed", zap.Error(err))
		return
	}
	zap.L().Debug("relay data success", zap.String("targetConn", targetConn.ID),
		zap.String("reqConn", conn.RemoteAddr().String()))
	targetConn.LastNormalActive = time.Now()
	relaySuccess = true
}

func (r *Relay) relay(targetConn *Connection, reqConn net.Conn, relayDataLen *int64) error {
	var errCH = make(chan error, 2)
	activelyTimeOut := false
	go func() {
		n, err := io.Copy(targetConn.Conn, reqConn)
		atomic.AddInt64(relayDataLen, int64(n))
		activelyTimeOut = true
		setErr := targetConn.Conn.SetReadDeadline(time.Unix(1136142245, 0))
		if setErr != nil {
			zap.L().Error("Failed to set read deadline", zap.Error(setErr))
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
		if !activelyTimeOut {
			if err != nil {
				errCH <- fmt.Errorf("targetConn -> reqConn: %w", err)
			} else {
				// reqConn.SetReadDeadline(time.Unix(1136142245, 0))
				errCH <- fmt.Errorf("relay dst actively disconnect")
			}
			return
		}
		errCH <- nil
		zap.L().Debug("targetConn -> reqConn success")
	}()
	zap.L().Debug("relay start", zap.String("targetConn", targetConn.Conn.RemoteAddr().String()),
		zap.String("reqConn", reqConn.RemoteAddr().String()))
	var err error
	for range 2 {
		err = <-errCH
		if err != nil {
			break
		}
	}
	if err != nil {
		return err
	}

	// reset read deadline to avoid read timeout
	setErr := targetConn.Conn.SetReadDeadline(time.Time{})
	if setErr != nil {
		zap.L().Error("Failed to reset read deadline", zap.Error(setErr))
	}
	return nil
}
