package protocol

type StatusCode int32

const (
	StatusError      StatusCode = 0
	StatusSuccess    StatusCode = -1
	StatusAuthFailed StatusCode = 1
)

type HandshakeReq struct {
	// SecretKeySelector is the selector of the secret key, 4 bytes use hex string(8 bytes in total)
	SecretKeySelector string `json:"secretKeySelector"`
	// AuthFieldB64 is encrypted with secret key,["AUTH"+RANDOM_STRING(16)]
	AuthFieldB64 string `json:"authFieldB64"`
	// AuthAAD is the additional authentication data
	AuthAAD string `json:"authAAD"`
	// EcdhPublicKey is the public key of the ECDH X25519 key exchange
	EcdhPublicKeyB64 string `json:"ecdhPublicKeyB64"`
}

type HandshakeResp struct {
	// RandomSharedKeyB64 string `json:"randomSharedKey"`
	Code StatusCode `json:"code"`
	Msg  string     `json:"msg"`
	// EcdhPublicKey is the public key of the ECDH X25519 key exchange
	EcdhPublicKeyB64 string `json:"ecdhPublicKeyB64"`
}

type ReqHead struct {
	Action  Action `json:"action"`
	DataLen int    `json:"dataLen"`
}

type RespHead struct {
	Code    StatusCode `json:"code"`
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
	ActionConnect Action = "connect"
	ActionPing    Action = "ping"
	ActionRelay   Action = "relay"
	// ActionClose is used to close the long connection
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
