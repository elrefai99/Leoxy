package config

import (
	"log"
	"os"

	"github.com/subosito/gotenv"
)

type Config struct {
	MAIN_SERVER    string
	PROXY_SERVER_1 string
	PROXY_SERVER_2 string
	PROXY_SERVER_3 string
	PROXY_SERVER_4 string
	RATE_LIMIT     string
}

func Load() (*Config, error) {
	if err := gotenv.Load(); err != nil {
		log.Fatal(".env file can't find")
	}

	

	var config *Config = &Config{
		MAIN_SERVER:    os.Getenv("MAIN_SERVER"),
		PROXY_SERVER_1: os.Getenv("PROXY_SERVER_1"),
		PROXY_SERVER_2: os.Getenv("PROXY_SERVER_2"),
		PROXY_SERVER_3: os.Getenv("PROXY_SERVER_3"),
		PROXY_SERVER_4: os.Getenv("PROXY_SERVER_4"),
		RATE_LIMIT:     os.Getenv("RATE_LIMIT"),
	}

	return config, nil
}
