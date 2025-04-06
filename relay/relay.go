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

	"github.com/doraemonkeys/WindSend-Relay/config"
	"github.com/doraemonkeys/WindSend-Relay/protocol"
	"github.com/doraemonkeys/WindSend-Relay/relay/auth"
	"github.com/doraemonkeys/doraemon/crypto"
	"go.uber.org/zap"
)

type Relay struct {
	config config.Config
	// nil when no secret keys
	auther *auth.Authentication
	// enableAuth    bool

	// ID -> Connection
	connections   map[string]*Connection
	connectionsMu sync.RWMutex
	keyConnLimit  map[string]*struct {
		count atomic.Int32
		limit int
	}
	keyConnLimitMu sync.RWMutex
}

type Connection struct {
	ID         string
	AuthkeyB64 string
	Cipher     crypto.SymmetricCipher

	// Lock when reading or writing
	Conn       net.Conn
	LastActive time.Time
	Relaying   bool
	Mu         sync.Mutex
}

// Be careful of deadlocks
func (c *Connection) SendMsgDetectAlive() (alive bool) {

	c.Mu.Lock()
	defer c.Mu.Unlock()

	l := zap.L().With(zap.String("id", c.ID), zap.String("addr", c.Conn.RemoteAddr().String()))

	err := protocol.SendHeartbeat(c.Conn, c.ID, c.Cipher)
	if err != nil {
		l.Error("sent heartbeat failed(detect alive)", zap.Error(err))
		return false
	}
	result := make(chan error, 1)
	go func() {
		head, err := protocol.ReadReqHead(c.Conn, c.Cipher)
		if err != nil {
			result <- err
		}
		if head.Action == protocol.ActionHeartbeat {
			result <- nil
		} else {
			result <- fmt.Errorf("unexpected action: %s", head.Action)
		}
	}()
	select {
	case err := <-result:
		if err != nil {
			l.Error("Failed to receive heartbeat", zap.Error(err))
			return false
		}
		c.LastActive = time.Now()
		return true
	case <-time.After(time.Second * 2):
		return false
	}
}

func NewRelay(config config.Config) *Relay {
	var secretKeys []string
	for _, secret := range config.SecretInfo {
		secretKeys = append(secretKeys, secret.SecretKey)
	}
	connLimit := make(map[string]*struct {
		count atomic.Int32
		limit int
	}, len(secretKeys))
	for _, secret := range config.SecretInfo {
		authKeyB64 := base64.StdEncoding.EncodeToString(auth.HashToAES192Key([]byte(secret.SecretKey)))
		connLimit[authKeyB64] = &struct {
			count atomic.Int32
			limit int
		}{count: atomic.Int32{}, limit: secret.MaxConn}
	}
	auther := auth.NewAuthentication(secretKeys)
	if len(secretKeys) == 0 {
		zap.L().Warn("No secret keys, authentication is disabled")
		auther = nil
	}
	if config.EnableAuth && len(secretKeys) == 0 {
		zap.L().Fatal("Enable authentication but no secret keys")
	}
	return &Relay{
		config:       config,
		auther:       auther,
		keyConnLimit: connLimit,
		connections:  make(map[string]*Connection),
	}
}

func (r *Relay) Run() {
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

func (r *Relay) mainProcess(conn net.Conn) {
	cipher, authKey, err := protocol.Handshake(conn, r.auther, r.config.EnableAuth)
	if err != nil {
		zap.L().Info("request handshake failed", zap.Error(err))
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

func (r *Relay) checkConnLimit(key string) bool {
	r.keyConnLimitMu.RLock()
	v, ok := r.keyConnLimit[key]
	r.keyConnLimitMu.RUnlock()
	if !ok {
		if r.auther != nil {
			panic("unknown key: " + key)
		}
		return true
	}
	return v.count.Load() < int32(v.limit)
}

func (r *Relay) handleConnect(conn net.Conn, head protocol.ReqHead, cipher crypto.SymmetricCipher, authKey auth.AES192Key) {
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

	authKeyB64 := base64.StdEncoding.EncodeToString(authKey)
	if !r.checkConnLimit(authKeyB64) {
		zap.L().Error("Too many connections", zap.String("id", req.ID))
		err = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
		if err != nil {
			zap.L().Error("Failed to send error", zap.Error(err))
		}
		return
	}

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
	r.connectionsMu.RLock()
	{
		if c, ok := r.connections[req.ID]; ok {
			r.connectionsMu.RUnlock()
			if c.Relaying || c.SendMsgDetectAlive() {
				zap.L().Error("Connection already exists", zap.String("id", req.ID))
				err = protocol.SendRespHeadError(conn, protocol.ActionConnect, "Connection already exists", cipher)
				if err != nil {
					zap.L().Error("Failed to send error", zap.Error(err))
				}
				return
			}

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

func (r *Relay) detectConnectionAlive() {
	for {
		time.Sleep(time.Second * 30)
		if len(r.connections) == 0 {
			continue
		}

		connections := make([]*Connection, 0, len(r.connections))
		r.connectionsMu.RLock()
		for _, c := range r.connections {
			connections = append(connections, c)
		}
		r.connectionsMu.RUnlock()

		for _, c := range connections {
			if c.Relaying {
				continue
			}
			c.Mu.Lock()
			err := protocol.SendHeartbeatNoResp(c.Conn, c.Cipher)
			if err != nil {
				zap.L().Info("detect connection alive failed", zap.Error(err), zap.String("id", c.ID),
					zap.String("addr", c.Conn.RemoteAddr().String()))
				r.RemoveLongConnection(c.ID)
			} else {
				c.LastActive = time.Now()
			}
			c.Mu.Unlock()
		}

	}
}

func (r *Relay) AddConnection(id string, conn net.Conn, authKey auth.AES192Key, cipher crypto.SymmetricCipher) {
	r.connectionsMu.Lock()
	c := &Connection{
		ID:         id,
		Conn:       conn,
		LastActive: time.Now(),
		Relaying:   false,
		AuthkeyB64: base64.StdEncoding.EncodeToString(authKey),
		Cipher:     cipher,
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
	r.keyConnLimitMu.RLock()
	v, ok := r.keyConnLimit[key]
	r.keyConnLimitMu.RUnlock()
	if !ok {
		if r.auther != nil {
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

	l := zap.L().With(zap.String("Action", "Relay"), zap.String("ReqAddr", conn.RemoteAddr().String()))
	req, err := protocol.ReadReq[protocol.RelayReq](conn, head.DataLen, cipher)
	if err != nil {
		l.Error("Failed to read relay request", zap.Error(err))
		return
	}

	l = l.With(zap.String("ID", req.ID))
	l.Info("Relay request")

	r.connectionsMu.RLock()
	targetConn, ok := r.connections[req.ID]
	r.connectionsMu.RUnlock()
	if !ok {
		l.Error("device not online")
		err := protocol.SendRespHeadError(conn, protocol.ActionRelay, "device not online", cipher)
		if err != nil {
			l.Error("Failed to send error", zap.Error(err))
		}
		return
	}
	// Simple processing, if targetConn is relaying, return an error
	if targetConn.Relaying {
		l.Error("Connection is already relaying")
		err := protocol.SendRespHeadError(conn, protocol.ActionRelay, "Connection is already relaying", cipher)
		if err != nil {
			l.Error("Failed to send error", zap.Error(err))
		}
		return
	}

	err = protocol.SendRespHeadOKWithMsg(conn, protocol.ActionRelay, "Relay start", cipher)
	if err != nil {
		l.Error("Failed to send relay start", zap.Error(err))
		return
	}

	targetConn.Mu.Lock()
	defer targetConn.Mu.Unlock()
	targetConn.Relaying = true
	defer func() {
		targetConn.Relaying = false
	}()
	err = protocol.SendRelayStart(targetConn.Conn, targetConn.Cipher)
	if err != nil {
		l.Error("Failed to send relay start", zap.Error(err))
		return
	}
	err = r.relay(targetConn, conn)
	if err != nil {
		l.Error("relay data failed", zap.Error(err))
		r.RemoveLongConnection(targetConn.ID)
		return
	}
	zap.L().Debug("relay data success", zap.String("targetConn", targetConn.ID),
		zap.String("reqConn", conn.RemoteAddr().String()))
	targetConn.LastActive = time.Now()

}

func (r *Relay) relay(targetConn *Connection, reqConn net.Conn) error {
	var errCH = make(chan error, 2)
	activelyTimeOut := false
	go func() {
		_, err := io.Copy(targetConn.Conn, reqConn)
		activelyTimeOut = true
		targetConn.Conn.SetReadDeadline(time.Unix(1136142245, 0))
		if err != nil {
			errCH <- fmt.Errorf("relay data to client: %w", err)
			return
		}
		errCH <- nil
		zap.L().Debug("reqConn -> targetConn success")
	}()
	go func() {
		_, err := io.Copy(reqConn, targetConn.Conn)
		if !activelyTimeOut {
			// reqConn.SetReadDeadline(time.Unix(1136142245, 0))
			errCH <- fmt.Errorf("relay dst active disconnect")
			return
		}
		if err != nil && !activelyTimeOut {
			errCH <- fmt.Errorf("relay data to server: %w", err)
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
	targetConn.Conn.SetReadDeadline(time.Time{})

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
	return nil
}
