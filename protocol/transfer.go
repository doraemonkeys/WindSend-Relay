package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/doraemonkeys/doraemon/crypto"
)

func ReadHandshakeReq(conn net.Conn) (HandshakeReq, error) {
	var itemBuf = make([]byte, 4)
	var item HandshakeReq

	var itemLen int32
	if _, err := io.ReadFull(conn, itemBuf[:4]); err != nil {
		return item, err
	}
	itemLen = int32(binary.LittleEndian.Uint32(itemBuf[:4]))
	// The head length cannot exceed 10KB to prevent memory overflow due to malicious attacks
	const maxItemLen = 1024 * 10
	if itemLen > maxItemLen {
		return item, fmt.Errorf("handshake len too large: %d", itemLen)
	}
	itemBuf = make([]byte, itemLen)
	if _, err := io.ReadFull(conn, itemBuf[:itemLen]); err != nil {
		return item, fmt.Errorf("read handshake failed, err: %w", err)
	}
	itemBuf = itemBuf[:itemLen]
	if err := json.Unmarshal(itemBuf, &item); err != nil {
		return item, fmt.Errorf("unmarshal handshake failed, err: %w", err)
	}
	return item, nil
}

func ReadReqHead(conn net.Conn, cipher crypto.SymmetricCipher) (ReqHead, error) {
	var itemBuf = make([]byte, 4)
	var item ReqHead

	var itemLen int32
	if _, err := io.ReadFull(conn, itemBuf[:4]); err != nil {
		return item, err
	}
	itemLen = int32(binary.LittleEndian.Uint32(itemBuf[:4]))
	// The head length cannot exceed 10KB to prevent memory overflow due to malicious attacks
	const maxItemLen = 1024 * 10
	if itemLen > maxItemLen {
		return item, fmt.Errorf("head len too large: %d", itemLen)
	}
	itemBuf = make([]byte, itemLen)
	if _, err := io.ReadFull(conn, itemBuf[:itemLen]); err != nil {
		return item, fmt.Errorf("read head failed, err: %w", err)
	}
	itemBuf = itemBuf[:itemLen]
	if cipher != nil {
		var err error
		itemBuf, err = cipher.Decrypt(itemBuf)
		if err != nil {
			return item, fmt.Errorf("decrypt head failed, err: %w", err)
		}
	}
	if err := json.Unmarshal(itemBuf, &item); err != nil {
		return item, fmt.Errorf("unmarshal head failed, err: %w", err)
	}
	return item, nil
}

func ReadReq[T any](conn net.Conn, dataLen int, cipher ...crypto.SymmetricCipher) (T, error) {
	var req T
	var reqBuf = make([]byte, dataLen)
	if _, err := io.ReadFull(conn, reqBuf[:dataLen]); err != nil {
		return req, fmt.Errorf("read req failed, err: %w", err)
	}
	reqBuf = reqBuf[:dataLen]
	if len(cipher) != 0 {
		var err error
		reqBuf, err = cipher[0].Decrypt(reqBuf)
		if err != nil {
			return req, fmt.Errorf("decrypt request failed, err: %w", err)
		}
	}
	if err := json.Unmarshal(reqBuf, &req); err != nil {
		return req, fmt.Errorf("req unmarshal failed, err: %w", err)
	}
	return req, nil
}

// sendStruct writes a struct to the connection.
//
//	|itemLen|item|
//	|4 bytes|itemLen bytes|
func sendStruct[T any](conn net.Conn, item T, cipher ...crypto.SymmetricCipher) error {
	respBuf, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("marshal item failed, err: %w", err)
	}
	// fmt.Println("send struct", string(respBuf))
	if len(cipher) != 0 {
		var err error
		respBuf, err = cipher[0].Encrypt(respBuf)
		if err != nil {
			return fmt.Errorf("encrypt item failed, err: %w", err)
		}
	}
	var itemLen = len(respBuf)

	var itemLenBuf [4]byte
	binary.LittleEndian.PutUint32(itemLenBuf[:], uint32(itemLen))
	if _, err := conn.Write(itemLenBuf[:]); err != nil {
		return fmt.Errorf("write item len failed, err: %w", err)
	}

	if _, err := conn.Write(respBuf); err != nil {
		return fmt.Errorf("write item failed, err: %w", err)
	}
	return nil
}

func sendReqHeadWithBody[T any](conn net.Conn, action Action, body T, cipher ...crypto.SymmetricCipher) error {
	jsonReq, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal req with body failed, err: %w", err)
	}
	if len(cipher) != 0 {
		var err error
		jsonReq, err = cipher[0].Encrypt(jsonReq)
		if err != nil {
			return fmt.Errorf("encrypt req with body failed, err: %w", err)
		}
	}

	var head ReqHead
	head.Action = action
	head.DataLen = len(jsonReq)
	err = sendStruct(conn, head, cipher...)
	if err != nil {
		return fmt.Errorf("send req head failed, err: %w", err)
	}
	_, err = conn.Write(jsonReq)
	if err != nil {
		return fmt.Errorf("write req with body failed, err: %w", err)
	}
	return nil
}

func SendHandshakeResp(conn net.Conn, resp HandshakeResp) error {
	return sendStruct(conn, resp)
}

func SendHeartbeatNoResp(conn net.Conn, cipher ...crypto.SymmetricCipher) error {
	var head ReqHead
	head.Action = ActionHeartbeat
	head.DataLen = 0
	return sendStruct(conn, head, cipher...)
}

func SendHeartbeat(conn net.Conn, id string, cipher ...crypto.SymmetricCipher) error {
	var req HeartbeatReq
	req.CommonReq.ID = id
	req.NeedResp = true

	return sendReqHeadWithBody(conn, ActionHeartbeat, req, cipher...)
}

func SendRespHeadOk(conn net.Conn, action Action, cipher ...crypto.SymmetricCipher) error {
	var head RespHead
	head.Code = StatusSuccess
	head.Msg = "OK"
	head.Action = action
	return sendStruct(conn, head, cipher...)
}

func SendRespHeadOKWithMsg(conn net.Conn, action Action, msg string, cipher ...crypto.SymmetricCipher) error {
	var head RespHead
	head.Code = StatusSuccess
	head.Msg = msg
	head.Action = action
	return sendStruct(conn, head, cipher...)
}

func SendRespHeadError(conn net.Conn, action Action, msg string, cipher ...crypto.SymmetricCipher) error {
	var head RespHead
	head.Code = StatusError
	head.Msg = msg
	head.Action = action
	return sendStruct(conn, head, cipher...)
}

func SendRelayStart(conn net.Conn, cipher ...crypto.SymmetricCipher) error {
	var head ReqHead
	head.Action = ActionRelay
	return sendStruct(conn, head, cipher...)
}
