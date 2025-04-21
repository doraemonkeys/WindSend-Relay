package relay

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/doraemonkeys/WindSend-Relay/server/protocol"
	"go.uber.org/zap"
)

func (r *Relay) detectConnectionAlive() {
	for {
		time.Sleep(time.Second * 60)
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
				if c.LastActive.Add(time.Hour * 6).Before(time.Now()) {
					zap.L().Error("unexpected: connection is relaying and timeout", zap.String("id", c.ID),
						zap.String("addr", c.Conn.RemoteAddr().String()))
					r.RemoveLongConnection(c.ID)
				}
				continue
			}
			err := c.detectAliveRandom()
			if err != nil {
				zap.L().Info("detect connection alive failed", zap.Error(err),
					zap.String("id", c.ID), zap.String("addr", c.Conn.RemoteAddr().String()))
				r.RemoveLongConnection(c.ID)
			}
		}
	}
}

func (c *Connection) detectAliveRandom() error {
	var err error
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if rand.IntN(10) == 0 {
		ok := c.sendMsgDetectAlive()
		if !ok {
			return fmt.Errorf("detect failed")
		}
	}
	err = protocol.SendHeartbeatNoResp(c.Conn, c.Cipher)
	if err != nil {
		return err
	}
	c.LastActive = time.Now()
	return nil
}
