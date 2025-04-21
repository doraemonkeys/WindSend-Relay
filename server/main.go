package main

import (
	"github.com/doraemonkeys/WindSend-Relay/server/admin"
	"github.com/doraemonkeys/WindSend-Relay/server/config"
	"github.com/doraemonkeys/WindSend-Relay/server/global"
	"github.com/doraemonkeys/WindSend-Relay/server/relay"
	"github.com/doraemonkeys/WindSend-Relay/server/storage"
)

func main() {
	cfg := config.ParseConfig()
	global.InitLogger(cfg.LogLevel)

	storage := storage.NewStorage(config.DBPath)
	relay := relay.NewRelay(*cfg, storage)

	adminServer := admin.NewAdminServer(relay, storage, &cfg.AdminConfig)
	go adminServer.Run()
	relay.Run()
}
