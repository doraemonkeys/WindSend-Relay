package protocol

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/doraemonkeys/WindSend-Relay/relay/auth"
	"github.com/doraemonkeys/doraemon/crypto"
)

func handshakeECDH(req HandshakeReq) (ecdhPublicKey *ecdh.PublicKey, shared auth.AES192Key, err error) {
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
	return sk.PublicKey(), auth.HashToAES192Key(sharedSecret), nil
}

func handleHandshakeReq(req HandshakeReq, auther *auth.Authentication) (resp *HandshakeResp, shared auth.AES192Key, authKey auth.AES192Key, err error) {
	// var authKey auth.AES192Key
	if auther != nil {
		ok, key := auther.Auth(req.SecretKeySelector, []byte(req.AuthField))
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
	}, shared, authKey, nil
}

// nil auther means no authentication,return nil authKey
func Handshake(conn net.Conn, auther *auth.Authentication) (cipher crypto.SymmetricCipher, authKey auth.AES192Key, err error) {
	req, err := ReadHandshakeReq(conn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read handshake request: %w", err)
	}
	resp, sharedKey, authKey, err := handleHandshakeReq(req, auther)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to handle handshake request: %w", err)
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
