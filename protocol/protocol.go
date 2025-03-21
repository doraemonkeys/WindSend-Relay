package protocol

type StatusCode int32

const (
	StatusError   StatusCode = 0
	StatusSuccess StatusCode = -1
)

type HandshakeReq struct {
	// SecretKeySelector is the selector of the secret key, 4 bytes use hex string(8 bytes in total)
	SecretKeySelector string `json:"secretKeySelector"`
	// AuthField is encrypted with secret key,["AUTH"+RANDOM_STRING(16)]
	AuthField string `json:"authField"`
	// EcdhPublicKey is the public key of the ECDH X25519 key exchange
	EcdhPublicKeyB64 string `json:"ecdhPublicKey"`
}

type HandshakeResp struct {
	// RandomSharedKeyB64 string `json:"randomSharedKey"`

	// EcdhPublicKey is the public key of the ECDH X25519 key exchange
	EcdhPublicKeyB64 string `json:"ecdhPublicKey"`
}

type ReqHead struct {
	Action  Action `json:"action"`
	DataLen int    `json:"dataLen"`
}

type RespHead struct {
	Code    StatusCode `json:"status"`
	Msg     string     `json:"msg"`
	Action  Action     `json:"action"`
	DataLen int        `json:"dataLen"`
}

type CommonReq struct {
	ID string `json:"id"`
}
type ConnectionReq struct {
	CommonReq
}

type Action string

const (
	ActionConnect   Action = "connect"
	ActionPing      Action = "ping"
	ActionRelay     Action = "relay"
	ActionClose     Action = "close"
	ActionHeartbeat Action = "heartbeat"
)

type HeartbeatReq struct {
	CommonReq
	NeedResp bool `json:"needResp"`
}

// type ConnectionResp struct {
// 	CommonResp
// }

type RelayReq struct {
	CommonReq
}

// type PingReq struct {
// 	CommonReq
// }

// type PingResp struct {
// 	CommonResp
// }
