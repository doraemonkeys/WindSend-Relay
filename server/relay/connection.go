package relay

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/doraemonkeys/WindSend-Relay/server/protocol"
	"github.com/doraemonkeys/doraemon/crypto"
	"go.uber.org/zap"
)

type Connection struct {
	ID         string
	AuthkeyB64 string
	Cipher     crypto.SymmetricCipher

	// Lock when reading or writing
	Conn             net.Conn
	LastNormalActive time.Time
	ConnectTime      time.Time
	// Lock immediately after locking, set to false after unlocking
	Relaying bool
	Mu       sync.Mutex
}

// Be careful of deadlocks
func (c *Connection) SendMsgDetectAlive() (alive bool) {

	c.Mu.Lock()
	defer c.Mu.Unlock()

	return c.sendMsgDetectAlive()
}

func (c *Connection) sendMsgDetectAlive() (alive bool) {

	l := zap.L().With(zap.String("id", c.ID), zap.String("addr", c.Conn.RemoteAddr().String()))

	err := protocol.SendHeartbeat(c.Conn, c.ID, c.Cipher)
	if err != nil {
		l.Warn("sent heartbeat failed(detect alive)", zap.Error(err))
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
		c.LastNormalActive = time.Now()
		return true
	case <-time.After(time.Second * 2):
		return false
	}
}
