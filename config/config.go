package config

import (
	"path/filepath"
	"gopkg.in/gcfg.v1"
)

type Config struct {
	RabbitMq struct {
		Host        string
		Username    string
		Password    string
		Port        string
		Vhost       string
		Compression bool
		Path        bool
	}
	Prefetch struct {
		Count  int
		Global bool
	}
	Exchange struct {
		Name       string
		Autodelete bool
		Type       string
		Durable    bool
	}
	Queue struct {
		Key  string
		Name string
		Max_Length int32
	}
	Deadexchange struct {
		Name       string
		AutoDelete bool
		Type       string
		Durable    bool
		Queue      string
		Retry      int
	}
	Logs struct {
		Error string
		Info  string
		Rpc   string
	}
	Output struct {
		Path string
	}
}

func LoadAndParse(location string) (*Config, error) {
	if !filepath.IsAbs(location) {
		location, err := filepath.Abs(location)

		if err != nil {
			return nil, err
		}

		location = location
	}

	cfg := Config{}
	if err := gcfg.ReadFileInto(&cfg, location); err != nil {
		return nil, err
	}

	return &cfg, nil
}
