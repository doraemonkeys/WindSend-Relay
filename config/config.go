package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
)

type SecretInfo struct {
	SecretKey string `json:"secret_key" env:"KEY,notEmpty"`
	MaxConn   int    `json:"max_conn" env:"MAX_CONN" envDefault:"5"`
}

type Config struct {
	ListenAddr  string       `json:"listen_addr" env:"WS_LISTEN_ADDR,notEmpty" envDefault:"0.0.0.0:16779"`
	MaxConn     int          `json:"max_conn" env:"WS_MAX_CONN" envDefault:"100"`
	IDWhitelist []string     `json:"id_whitelist" envPrefix:"WS_ID_WHITELIST"`
	SecretInfo  []SecretInfo `json:"secret_info" envPrefix:"WS_SECRET"`
}

func ParseConfig() *Config {
	configFile := flag.String("config", "", "config file, other command line args will be ignored")
	useEnv := flag.Bool("use-env", false, "use env, other command line args will be ignored")

	var config Config
	flag.StringVar(&config.ListenAddr, "listen-addr", "0.0.0.0:16779", "listen address")
	flag.IntVar(&config.MaxConn, "max-conn", 100, "max connection")

	flag.Parse()

	if *useEnv {
		return parseEnv()
	}

	if *configFile != "" {
		jsonFile, err := os.Open(*configFile)
		if err != nil {
			zap.L().Error("Failed to open config file", zap.Error(err))
		}
		defer jsonFile.Close()

		json.NewDecoder(jsonFile).Decode(&config)
	}

	return &config
}

func parseEnv() *Config {
	var config, err = env.ParseAs[Config]()
	if err != nil {
		zap.L().Error("Failed to parse env", zap.Error(err))
	}
	return &config
}
