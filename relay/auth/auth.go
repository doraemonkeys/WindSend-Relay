package auth

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/doraemonkeys/WindSend-Relay/tool"
	"github.com/doraemonkeys/doraemon"
	"github.com/doraemonkeys/doraemon/crypto"
	"go.uber.org/zap"
)

type Authentication struct {
	RawKeyList   []string
	KeySelectors map[string][]tool.AES192Key
	selectorMu   sync.RWMutex
}

func NewAuthentication(keys []string) *Authentication {
	zap.L().Info("Server secret key count", zap.Int("count", len(keys)))
	selectors := make(map[string][]tool.AES192Key, len(keys))
	for i, key := range keys {
		aesKey := tool.HashToAES192Key([]byte(key))
		selector := getAES192KeySelector(aesKey)
		ks, ok := selectors[selector]
		if !ok {
			ks = make([]tool.AES192Key, 0, 1)
		}
		for _, k := range ks {
			if bytes.Equal(k, aesKey) {
				continue
			}
		}
		zap.L().Debug(fmt.Sprintf("secret key: %d, key: %s", i, key))
		zap.L().Debug(fmt.Sprintf("selector: %s, aesKey: %s", selector, hex.EncodeToString(aesKey)))
		selectors[selector] = append(ks, aesKey)
	}
	return &Authentication{
		RawKeyList:   keys,
		KeySelectors: selectors,
	}
}

// return 4 bytes hash prefix encoded in hex
func getAES192KeySelector(key tool.AES192Key) string {
	hash, err := doraemon.ComputeSHA256Hex(bytes.NewReader(key))
	if err != nil {
		panic("unreachable")
	}
	return hash[:8]
}

func (a *Authentication) Auth(selector string, authField []byte, additionalData ...[]byte) (bool, tool.AES192Key) {
	a.selectorMu.RLock()
	ks, ok := a.KeySelectors[selector]
	a.selectorMu.RUnlock()
	if !ok {
		return false, nil
	}
	for _, k := range ks {
		cipher, err := crypto.NewAESGCM(k)
		if err != nil {
			panic("unreachable: Invalid AES192Key " + err.Error())
		}
		plaintext, err := cipher.DecryptAuth(authField, additionalData...)
		if err != nil {
			continue
		}
		if bytes.HasPrefix(plaintext, []byte("AUTH")) {
			return true, k
		}
	}
	return false, nil
}
