package utils

import (
	"fmt"

	"github.com/spf13/viper"
)

var configPath string
var configs map[string]*viper.Viper

func init() {
	configs = make(map[string]*viper.Viper)
	configPath = "config/"
}

func GetConfig(filename string) (*viper.Viper, error) {
	if val, ok := configs[filename]; ok {
		return val, nil
	}

	v := viper.New()
	v.AddConfigPath(configPath)
	v.SetConfigName(filename)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	configs[filename] = v
	return v, nil
}

func GetString(filename string, key string) string {
	config, err := GetConfig(filename)
	if err != nil {
		return ""
	}
	if config == nil {
		return ""
	}

	return config.GetString(key)
}

func GetInt(filename string, key string) int {
	config, err := GetConfig(filename)
	if err != nil {
		return 0
	}
	if config == nil {
		return 0
	}

	return config.GetInt(key)
}

func GetLocalAddress() string {
	host := GetString("app", "host")
	port := GetInt("app", "port")
	address := fmt.Sprintf("%s:%d", host, port)
	return address
}
