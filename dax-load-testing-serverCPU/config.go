package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed config.json
var configData []byte

type ClientConfig struct {
	ConnectionTimeout     int64 `json:"ConnectionTimeout"`
	RequestTimeout        int64 `json:"RequestTimeout"`
	ReadRetries           int64 `json:"ReadRetries"`
	WriteRetries          int64 `json:"WriteRetries"`
	MaxConcurrency        int64 `json:"MaxConcurrency"`
	MaxPendingConnections int64 `json:"MaxPendingConnections"`
}

type TrafficConfig struct {
	ItemSizes          map[string]int64 `json:"ItemSizes"`
	NumberOfAttributes int64            `json:"NumberOfAttributes"`
	APIs               []string         `json:"APIs"`
	CacheHitPercentage int64            `json:"CacheHitPercentage"`
}

type TestConfig struct {
	Name          string `json:"Name"`
	Description   string `json:"Description"`
	ClientConfig  string `json:"ClientConfig"`
	Reboot        int64  `json:"Reboot"`
	TrafficConfig string `json:"TrafficConfig"`
}

type Config struct {
	ClientConfigs  map[string]ClientConfig  `json:"ClientConfigs"`
	TrafficConfigs map[string]TrafficConfig `json:"TrafficConfigs"`
	TestConfigs    map[string]TestConfig    `json:"TestConfigs"`
	Table          string                   `json:"Table"`
}

type AppConfig struct {
	ClientConfig  ClientConfig
	TrafficConfig TrafficConfig
	TestConfig    TestConfig
	Table         string
}

func getAppConfig(f *flags) *AppConfig {
	cfg := &Config{}

	if err := json.Unmarshal(configData, cfg); err != nil {
		panic(err)
	}

	var ok bool
	app := &AppConfig{}

	app.TestConfig, ok = cfg.TestConfigs[f.test]
	if !ok {
		panic(fmt.Sprintf("unable to get test config: %s", f.test))
	}

	app.ClientConfig, ok = cfg.ClientConfigs[app.TestConfig.ClientConfig]
	if !ok {
		panic(fmt.Sprintf("unable to get client config: %s", app.TestConfig.ClientConfig))
	}

	app.TrafficConfig, ok = cfg.TrafficConfigs[app.TestConfig.TrafficConfig]
	if !ok {
		panic(fmt.Sprintf("unable to get traffic config: %s", app.TestConfig.TrafficConfig))
	}
	app.Table = cfg.Table

	fmt.Println(app.TrafficConfig.APIs)
	if !inArray(app.TrafficConfig.APIs, f.op) && (f.op != "Write" && f.op != "write" && f.op != "Read" && f.op != "read") {
		panic(fmt.Sprintf("unable to get op: %s", f.op))
	}

	return app
}
