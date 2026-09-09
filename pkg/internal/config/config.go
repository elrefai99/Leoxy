package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Upstream struct {
	Name      string `mapstructure:"name"`
	Path      string `mapstructure:"path"`
	IP        bool   `mapstructure:"ip"`
	ServerURL string `mapstructure:"server_url"`
}

type ServerConfig struct {
	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`
	Upstream []Upstream `mapstructure:"upstream"`
}

var Config *ServerConfig

func Load() (*ServerConfig, error) {
	viper.AddConfigPath("leoxy")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	err := viper.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %s", err)
	}

	var cfg ServerConfig
	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}

	Config = &cfg
	return Config, nil
}
