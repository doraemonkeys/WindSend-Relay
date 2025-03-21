package relay

import (
	"encoding/base64"
	"fmt"
	"io"
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

//	type Config struct {
//	    ListenAddr  string   `json:"listen_addr" env:"WS_LISTEN_ADDR,notEmpty" envDefault:"0.0.0.0:16779"`
//	    MaxConn     int      `json:"max_conn" env:"WS_MAX_CONN" envDefault:"100"`
//	    IDWhitelist []string `json:"id_whitelist" envPrefix:"WS_ID_WHITELIST"`
//	}
type Relay struct {
	config        config.Config
	auther        *auth.Authentication
	connections   map[string]*Connection
	connectionsMu sync.RWMutex
	keyConnLimit  map[string]*struct {
		count atomic.Int32
		limit int
	}
	// keyConnLimitMu sync.RWMutex
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

func (c *Connection) DetectAlive() (alive bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	err := protocol.SendHeartbeat(c.Conn, true, c.Cipher)
	if err != nil {
		zap.L().Error("Failed to send heartbeat", zap.Error(err))
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
			zap.L().Error("Failed to receive heartbeat", zap.Error(err), zap.String("addr", c.Conn.RemoteAddr().String()),
				zap.String("id", c.ID))
			return false
		}
		c.LastActive = time.Now()
		return true
	case <-time.After(time.Second * 2):
		return false
	}
}

// type SecretInfo struct {
// 	SecretKey string `json:"secret_key" env:"KEY,notEmpty"`
// 	MaxConn   int    `json:"max_conn" env:"MAX_CONN" envDefault:"5"`
// }

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
		connLimit[secret.SecretKey] = &struct {
			count atomic.Int32
			limit int
		}{count: atomic.Int32{}, limit: secret.MaxConn}
	}
	auther := auth.NewAuthentication(secretKeys)
	return &Relay{
		config:       config,
		auther:       auther,
		keyConnLimit: connLimit,
	}
}

func (r *Relay) Run() {
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
	cipher, authKey, err := protocol.Handshake(conn, r.auther)
	if err != nil {
		zap.L().Error("Failed to handshake", zap.Error(err))
		return
	}
	head, err := protocol.ReadReqHead(conn, cipher)
	if err != nil {
		zap.L().Error("Failed to read common request head", zap.Error(err))
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
		_ = protocol.SendHeadError(conn, head.Action, "Unknown action")
		_ = conn.Close()
	}
}

func (r *Relay) handleConnect(conn net.Conn, head protocol.ReqHead, cipher crypto.SymmetricCipher, authKey auth.AES192Key) {
	req, err := protocol.ReadReq[protocol.ConnectionReq](conn, head.DataLen, cipher)
	if err != nil {
		_ = conn.Close()
		zap.L().Error("Failed to read connection request", zap.Error(err))
		return
	}
	zap.L().Info("Connection request", zap.String("id", req.ID))
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
			r.connectionsMu.Unlock()
			if c.DetectAlive() {
				zap.L().Error("Connection already exists", zap.String("id", req.ID))
				err = protocol.SendHeadError(conn, protocol.ActionConnect, "Connection already exists", cipher)
				if err != nil {
					zap.L().Error("Failed to send error", zap.Error(err))
				}
				_ = conn.Close()
				return
			}

			r.RemoveConnection(req.ID)

			r.connectionsMu.RLock()
		}
		if len(r.connections) >= r.config.MaxConn {
			r.connectionsMu.RUnlock()
			zap.L().Error("Too many connections", zap.String("id", req.ID))
			err = protocol.SendHeadError(conn, protocol.ActionConnect, "Too many connections", cipher)
			if err != nil {
				zap.L().Error("Failed to send error", zap.Error(err))
			}
			_ = conn.Close()
			return
		}
	}
	r.connectionsMu.RUnlock()

	r.AddConnection(req.ID, conn, authKey)

	err = protocol.SendHeadOk(conn, protocol.ActionConnect, cipher)
	if err != nil {
		zap.L().Error("Failed to send OK", zap.Error(err), zap.String("id", req.ID),
			zap.String("addr", conn.RemoteAddr().String()))
		r.RemoveConnection(req.ID)
		return
	}
}

func (r *Relay) detectConnectionAlive(id string) {
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
			c.Mu.Lock()
			err := protocol.SendHeartbeat(c.Conn, false, c.Cipher)
			if err != nil {
				r.RemoveConnection(c.ID)
			}
			c.Mu.Unlock()
		}

	}
}

func (r *Relay) AddConnection(id string, conn net.Conn, authKey auth.AES192Key) {
	r.connectionsMu.Lock()
	c := &Connection{
		ID:         id,
		Conn:       conn,
		LastActive: time.Now(),
		Relaying:   false,
		AuthkeyB64: base64.StdEncoding.EncodeToString(authKey),
	}
	r.connections[id] = c
	r.connectionsMu.Unlock()
	r.addKeyConnCount(c.AuthkeyB64, 1)
}

func (r *Relay) RemoveConnection(id string) {
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
	// r.keyConnLimitMu.RLock()
	v, ok := r.keyConnLimit[key]
	// r.keyConnLimitMu.RUnlock()
	if !ok {
		panic("unknown key: " + key)
	}
	return v.count.Add(add)
}

func (r *Relay) handlePing(conn net.Conn, _ protocol.ReqHead, cipher crypto.SymmetricCipher) {
	defer conn.Close()

	l := zap.L().With(zap.String("Action", "Ping"), zap.String("addr", conn.RemoteAddr().String()))
	l.Info("Ping request")
	err := protocol.SendHeadOk(conn, protocol.ActionPing, cipher)
	if err != nil {
		l.Error("Failed to send OK", zap.Error(err))
		return
	}
}

func (r *Relay) handleRelay(conn net.Conn, head protocol.ReqHead, cipher crypto.SymmetricCipher) {
	defer conn.Close()

	l := zap.L().With(zap.String("Action", "Relay"), zap.String("ReqAddr", conn.RemoteAddr().String()))
	l.Info("Relay request")
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
		l.Error("Connection not found", zap.String("ID", req.ID))
		err := protocol.SendHeadError(conn, protocol.ActionRelay, "Connection not found", cipher)
		if err != nil {
			l.Error("Failed to send error", zap.Error(err))
		}
		return
	}
	if targetConn.Relaying {
		l.Warn("Connection is already relaying", zap.String("ID", req.ID))
	}

	targetConn.Mu.Lock()
	defer targetConn.Mu.Unlock()
	targetConn.Relaying = true
	err = r.relay(targetConn.Conn, conn)
	if err != nil {
		l.Error("relay data failed", zap.Error(err))
	}
	targetConn.LastActive = time.Now()
	targetConn.Relaying = false
}

func (r *Relay) relay(targetConn net.Conn, reqConn net.Conn) error {
	var errCH = make(chan error, 2)
	go func() {
		_, err := io.Copy(targetConn, reqConn)
		if err != nil {
			errCH <- fmt.Errorf("failed to relay data to client: %w", err)
		}
	}()
	go func() {
		_, err := io.Copy(reqConn, targetConn)
		if err != nil {
			errCH <- fmt.Errorf("failed to relay data to server: %w", err)
		}
	}()
	var err error
	for range 2 {
		err = <-errCH
		if err != nil {
			break
		}
	}
	return err
}
