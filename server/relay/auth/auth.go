package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/doraemonkeys/WindSend-Relay/server/tool"
	"github.com/doraemonkeys/doraemon"
	"github.com/doraemonkeys/doraemon/crypto"
	"go.uber.org/zap"
)

type Authentication struct {
	RawKeyList        []string
	KeySelectors      map[string][]tool.AES192Key
	selectorMu        sync.RWMutex
	rawKeyToAES192Key map[string]tool.AES192Key
	randomSalt        []byte
}

func NewAuthentication(keys []string) *Authentication {
	zap.L().Info("Server secret key count", zap.Int("count", len(keys)))
	randomSalt := make([]byte, 12)
	_, err := rand.Read(randomSalt)
	if err != nil {
		panic("unreachable: " + err.Error())
	}
	zap.L().Debug("random salt", zap.String("salt", base64.StdEncoding.EncodeToString(randomSalt)))

	rawKeyToAES192Key := make(map[string]tool.AES192Key, len(keys))
	selectors := make(map[string][]tool.AES192Key, len(keys))
	for i, key := range keys {
		aesKey := tool.AES192KeyKDF(key, randomSalt)
		rawKeyToAES192Key[key] = aesKey
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
		RawKeyList:        keys,
		KeySelectors:      selectors,
		randomSalt:        randomSalt,
		rawKeyToAES192Key: rawKeyToAES192Key,
	}
}

func (a *Authentication) GetRandomSalt() []byte {
	return a.randomSalt
}

func (a *Authentication) GetSaltB64() string {
	return base64.StdEncoding.EncodeToString(a.randomSalt)
}

func (a *Authentication) GetAllAuthKeys() map[string]tool.AES192Key {
	return a.rawKeyToAES192Key
}

// return 4 bytes hash prefix encoded in hex
func getAES192KeySelector(key tool.AES192Key) string {
	hash := doraemon.ComputeSHA256Hex(bytes.NewReader(key)).Unwrap()
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
