package dto

import "time"

type PageInfo struct {
	// To retrive all items, just set the page very large
	Page     int `json:"page" form:"page" binding:"required,min=1"`
	PageSize int `json:"pageSize" form:"pageSize" binding:"required,min=1,max=100"`
}

type PageInfoSort struct {
	// SortBy: field name, empty string means no sort
	SortBy string `json:"sortBy" form:"sortBy"`
	// SortType: "asc" or "desc".
	SortType string `json:"sortType" form:"sortType" binding:"omitempty,oneof=asc desc"`
}

type PaginatedData[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type HistoryStatistic struct {
	TotalRelayCount        int   `json:"totalRelayCount"`
	TotalRelayErrCount     int   `json:"totalRelayErrCount"`
	TotalRelayOfflineCount int   `json:"totalRelayOfflineCount"`
	TotalRelayMs           int64 `json:"totalRelayMs"`
	TotalRelayBytes        int64 `json:"totalRelayBytes"`
}

type ActiveConnection struct {
	ID          string           `json:"id"`
	ReqAddr     string           `json:"reqAddr"`
	ConnectTime time.Time        `json:"connectTime"`
	LastActive  time.Time        `json:"lastActive"`
	Relaying    bool             `json:"relaying"`
	History     HistoryStatistic `json:"history"`
}

type ReqHistoryStatistic struct {
	PageInfo
	PageInfoSort
}

// sort by TotalRelayCount, TotalRelayMs, TotalRelayBytes
type RespHistoryStatistic = PaginatedData[HistoryStatistic]
