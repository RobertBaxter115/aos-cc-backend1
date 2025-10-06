package config

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all configuration for the provisioning service
type Config struct {
	Server     ServerConfig     `koanf:"server"`
	Redis      RedisConfig      `koanf:"redis"`
	NodeAPI    NodeAPIConfig    `koanf:"node_api"`
	Prediction PredictionConfig `koanf:"prediction"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port int `koanf:"port"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr     string `koanf:"addr"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
}

// NodeAPIConfig holds Node Management API configuration
type NodeAPIConfig struct {
	BaseURL string        `koanf:"base_url"`
	Timeout time.Duration `koanf:"timeout"`
}

// PredictionConfig holds prediction algorithm configuration
type PredictionConfig struct {
	ActivityWindow         time.Duration `koanf:"activity_window"`
	ActivityThreshold      int           `koanf:"activity_threshold"`
	PredictionWindow       time.Duration `koanf:"prediction_window"`
	MinReadyNodes          int           `koanf:"min_ready_nodes"`
	MaxReadyNodes          int           `koanf:"max_ready_nodes"`
	IdleTerminationTimeout time.Duration `koanf:"idle_termination_timeout"`
	BootingNodeTimeout     time.Duration `koanf:"booting_node_timeout"`
	ScalingCheckInterval   time.Duration `koanf:"scaling_check_interval"`
}

// Load loads configuration from environment variables and optional config file
func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Load from config file if provided
	if configPath != "" {
		if err := k.Load(file.Provider(configPath), json.Parser()); err != nil {
			return nil, fmt.Errorf("error loading config file: %w", err)
		}
	}

	// Load from environment variables (with prefix)
	if err := k.Load(env.Provider("APP_", ".", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading env vars: %w", err)
	}

	// Set defaults
	setDefaults(k)

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(k *koanf.Koanf) {
	// Server defaults
	k.Set("server.port", 8081)

	// Redis defaults
	if k.String("redis.addr") == "" {
		k.Set("redis.addr", "localhost:6379")
	}
	if k.String("redis.password") == "" {
		k.Set("redis.password", "")
	}
	if k.Int("redis.db") == 0 {
		k.Set("redis.db", 0)
	}

	// Node API defaults
	if k.String("node_api.base_url") == "" {
		k.Set("node_api.base_url", "http://localhost:8080")
	}
	if k.Duration("node_api.timeout") == 0 {
		k.Set("node_api.timeout", 10*time.Second)
	}

	// Prediction defaults
	if k.Duration("prediction.activity_window") == 0 {
		k.Set("prediction.activity_window", 2*time.Minute)
	}
	if k.Int("prediction.activity_threshold") == 0 {
		k.Set("prediction.activity_threshold", 3)
	}
	if k.Duration("prediction.prediction_window") == 0 {
		k.Set("prediction.prediction_window", 1*time.Minute)
	}
	if k.Int("prediction.min_ready_nodes") == 0 {
		k.Set("prediction.min_ready_nodes", 1)
	}
	if k.Int("prediction.max_ready_nodes") == 0 {
		k.Set("prediction.max_ready_nodes", 5)
	}
	if k.Duration("prediction.idle_termination_timeout") == 0 {
		k.Set("prediction.idle_termination_timeout", 5*time.Minute)
	}
	if k.Duration("prediction.booting_node_timeout") == 0 {
		k.Set("prediction.booting_node_timeout", 2*time.Minute)
	}
	if k.Duration("prediction.scaling_check_interval") == 0 {
		k.Set("prediction.scaling_check_interval", 10*time.Second)
	}
}
