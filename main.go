package main

import (
	"github.com/doraemonkeys/WindSend-Relay/config"
	"github.com/doraemonkeys/WindSend-Relay/global"
	"github.com/doraemonkeys/WindSend-Relay/relay"
	"github.com/doraemonkeys/WindSend-Relay/storage"
)

func main() {
	cfg := config.ParseConfig()
	global.InitLogger(cfg.LogLevel)

	storage := storage.NewStorage(config.DBPath)
	relay := relay.NewRelay(*cfg, storage)

	relay.Run()
}
