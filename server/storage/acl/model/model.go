package model

import (
	"time"

	"gorm.io/gorm"
)

// type RelayStatistic struct {
// 	TotalRelayCount        int   `json:"total_relay_count"`
// 	TotalRelayErrCount     int   `json:"total_relay_err_count"`
// 	TotalRelayOfflineCount int   `json:"total_relay_offline_count"`
// 	TotalRelayMs           int64 `json:"total_relay_ms"`
// 	TotalRelayBytes        int64 `json:"total_relay_bytes"`
// 	FirstRelayTime         int64 `json:"first_relay_time"`
// 	LastRelayTime          int64 `json:"last_relay_time"`
// }

type RelayStatistic struct {
	ID                     string `gorm:"column:id;primary_key;index"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
	TotalRelayCount        int   `gorm:"column:total_relay_count;default:0"`
	TotalRelayErrCount     int   `gorm:"column:total_relay_err_count;default:0"`
	TotalRelayOfflineCount int   `gorm:"column:total_relay_offline_count;default:0"`
	TotalRelayMs           int64 `gorm:"column:total_relay_ms;default:0"`
	TotalRelayBytes        int64 `gorm:"column:total_relay_bytes;default:0"`
}

type KeyValue struct {
	gorm.Model
	Key   string `gorm:"column:key;unique;not null;index"`
	Value string `gorm:"column:value;not null;default:''"`
}
