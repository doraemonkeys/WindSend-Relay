package protocol

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/doraemonkeys/WindSend-Relay/relay/auth"
	"github.com/doraemonkeys/WindSend-Relay/tool"
	"github.com/doraemonkeys/doraemon/crypto"
)

func handshakeECDH(req HandshakeReq) (ecdhPublicKey *ecdh.PublicKey, shared tool.AES192Key, err error) {
	publicKey, err := base64.StdEncoding.DecodeString(req.EcdhPublicKeyB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode public key: %w", err)
	}
	remotePk, err := ecdh.X25519().NewPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create public key: %w", err)
	}
	curve := ecdh.X25519()
	sk, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key: %w", err)
	}
	sharedSecret, err := sk.ECDH(remotePk)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate shared secret: %w", err)
	}
	return sk.PublicKey(), tool.HashToAES192Key(sharedSecret), nil
}

func handleHandshakeReq(req HandshakeReq, authenticator *auth.Authentication, enableAuth bool) (resp *HandshakeResp, shared tool.AES192Key, authKey tool.AES192Key, err error) {
	if req.AuthFieldB64 != "" && authenticator == nil {
		// Relay server has no configured keys, but the client sent an authentication message
		return nil, nil, nil, fmt.Errorf("invalid handshake request: invalid auth field")
	}
	if enableAuth && req.AuthFieldB64 == "" {
		return nil, nil, nil, fmt.Errorf("invalid handshake request: no auth field")
	}
	if req.AuthFieldB64 != "" && req.AuthAAD == "" {
		// Force the client to send the auth aad
		return nil, nil, nil, fmt.Errorf("invalid handshake request: no auth aad")
	}
	// As long as AuthFieldB64 is not empty, authentication is performed
	if req.AuthFieldB64 != "" && authenticator != nil {
		authField, err := base64.StdEncoding.DecodeString(req.AuthFieldB64)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to decode auth field: %w", err)
		}
		ok, key := authenticator.Auth(req.SecretKeySelector, authField, []byte(req.AuthAAD))
		if !ok {
			return nil, nil, nil, fmt.Errorf("failed to authenticate")
		}
		authKey = key
	}
	ecdhPublicKey, shared, err := handshakeECDH(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("handshake ECDH: %w", err)
	}
	ecdhPublicKeyBytes := ecdhPublicKey.Bytes()
	if authKey != nil {
		cipher, err := crypto.NewAESGCM(authKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create AESGCM: %w", err)
		}
		encrypted, err := cipher.EncryptAuth(ecdhPublicKeyBytes, []byte("AUTH"))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to encrypt ecdh public key: %w", err)
		}
		ecdhPublicKeyBytes = encrypted
	}
	return &HandshakeResp{
		EcdhPublicKeyB64: base64.StdEncoding.EncodeToString(ecdhPublicKeyBytes),
		Code:             StatusSuccess,
	}, shared, authKey, nil
}

// nil authenticator means no authentication,return nil authKey
func Handshake(conn net.Conn, authenticator *auth.Authentication, enableAuth bool) (cipher crypto.SymmetricCipher, authKey tool.AES192Key, err error) {
	req, err := ReadHandshakeReq(conn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read handshake request: %w", err)
	}
	resp, sharedKey, authKey, err := handleHandshakeReq(req, authenticator, enableAuth)
	if err != nil {
		_ = SendHandshakeResp(conn, HandshakeResp{
			Code: StatusAuthFailed,
			Msg:  err.Error(),
		})
		return nil, nil, fmt.Errorf("handle handshake request: %w", err)
	}
	err = SendHandshakeResp(conn, *resp)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send handshake response: %w", err)
	}
	cipher, err = crypto.NewAESGCM(sharedKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AESGCM: %w", err)
	}
	return cipher, authKey, nil
}

// func randomAES192Key() AES192Key {
// 	key := make(AES192Key, 192/8)
// 	_, err := rand.Read(key)
// 	if err != nil {
// 		panic("unreachable: Failed to generate random AES192Key " + err.Error())
// 	}
// 	return key
// }
