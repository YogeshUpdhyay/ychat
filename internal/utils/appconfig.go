package utils

import (
	"os"
	"path/filepath"

	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Port           string `yaml:"port"`
	Name           string `yaml:"server_name"`
	Version        string `yaml:"version"`
	StreamProtocol string `yaml:"stream_protocol"`
	DatabasePath   string `yaml:"database_path"`
	RelayAddr      string `yaml:"relay_addr"`
}

func DefaultAppConfig() AppConfig {
	return AppConfig{
		Port:           constants.ServerPortDefault,
		Name:           constants.ServerNameDefault,
		Version:        constants.ServerVersionDefault,
		StreamProtocol: constants.StreamProtocolDefault,
		DatabasePath:   constants.DatabasePathDefault,
		RelayAddr:      constants.DefaultRelayAddr,
	}
}

// WriteConfigYML writes the default config to the default path from constants
func WriteAppConfig() error {
	config := DefaultAppConfig()
	dir := constants.ApplicationDataDir
	path := filepath.Join(dir, constants.ApplicationConfigFileName)

	// if config file already exists, do nothing
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	// ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	return encoder.Encode(config)
}

// ReadConfigYML reads the config from the default path in constants

func GetAppConfig() AppConfig {
	var config AppConfig
	path := filepath.Join(constants.ApplicationDataDir, constants.ApplicationConfigFileName)
	file, err := os.Open(path)
	if err != nil {
		return DefaultAppConfig()
	}
	defer file.Close()
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)

	if err != nil {
		return DefaultAppConfig()
	}

	return config
}
