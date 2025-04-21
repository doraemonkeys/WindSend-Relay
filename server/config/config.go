package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v11"
	"github.com/doraemonkeys/WindSend-Relay/server/version"
	"github.com/doraemonkeys/doraemon"
)

const DBPath = "data/relay.db"
const WebStaticDir = "static/web"

type SecretInfo struct {
	SecretKey string `json:"secret_key" env:"KEY,notEmpty"`
	MaxConn   int    `json:"max_conn" env:"MAX_CONN" envDefault:"5"`
}

type Config struct {
	ListenAddr  string       `json:"listen_addr" env:"WS_LISTEN_ADDR,notEmpty" envDefault:"0.0.0.0:16779"`
	MaxConn     int          `json:"max_conn" env:"WS_MAX_CONN" envDefault:"100"`
	IDWhitelist []string     `json:"id_whitelist" envPrefix:"WS_ID_WHITELIST"`
	SecretInfo  []SecretInfo `json:"secret_info" envPrefix:"WS_SECRET"`
	EnableAuth  bool         `json:"enable_auth" env:"WS_ENABLE_AUTH" envDefault:"false"`
	LogLevel    string       `json:"log_level" env:"WS_LOG_LEVEL" envDefault:"INFO"`
	AdminConfig AdminConfig  `json:"admin_config" envPrefix:"WS_ADMIN"`
}

type AdminConfig struct {
	User     string `json:"user" env:"USER" envDefault:"admin"`
	Password string `json:"password" env:"PASSWORD" envDefault:""`
	Addr     string `json:"addr" env:"ADDR" envDefault:"0.0.0.0:16780"`
	// JWTSecret string `json:"jwt_secret" env:"JWT_SECRET" envDefault:""`
}

func ParseConfig() *Config {
	configFile := flag.String("config", "", "json config file, other command line args will be ignored")
	useEnv := flag.Bool("use-env", false, "use env, other command line args will be ignored")

	var config Config
	flag.StringVar(&config.ListenAddr, "listen-addr", "0.0.0.0:16779", "listen address")
	flag.IntVar(&config.MaxConn, "max-conn", 100, "max connection")
	flag.StringVar(&config.LogLevel, "log-level", "INFO", "log level")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Println("WindSend-Relay", "v"+version.Version)
		fmt.Println("BuildTime:", version.BuildTime)
		fmt.Println("BuildHash:", version.BuildHash)
		os.Exit(0)
	}

	defer amendConfig(&config)

	if *useEnv {
		log.Println("parse config from env")
		return parseEnv()
	}

	if *configFile != "" {
		log.Println("parse config from file", *configFile)
		config.AdminConfig.Addr = "0.0.0.0:16780"
		config.AdminConfig.User = "admin"
		jsonFile, err := os.Open(*configFile)
		if err != nil {
			log.Fatal("Failed to open config file", err)
		}
		defer jsonFile.Close()

		json.NewDecoder(jsonFile).Decode(&config)
		return &config
	}

	log.Println("parse config from command line")
	return &config
}

func amendConfig(config *Config) {
	if config.AdminConfig.User == "" {
		config.AdminConfig.User = "admin"
		log.Println("generated admin user", config.AdminConfig.User)
	}
	const adminPasswordLength = 12
	if config.AdminConfig.Password == "" {
		config.AdminConfig.Password = doraemon.GenRandomAsciiString(adminPasswordLength)
		log.Println("generated admin password", config.AdminConfig.Password)
	}
	if len(config.AdminConfig.Password) < adminPasswordLength {
		config.AdminConfig.Password = doraemon.GenRandomAsciiString(adminPasswordLength)
		log.Fatal("password must be at least", adminPasswordLength, "characters")
	}
}

func parseEnv() *Config {
	var config, err = env.ParseAs[Config]()
	if err != nil {
		log.Fatal("Failed to parse env", err)
	}
	return &config
}
