package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type ConsistencyMode string

const (
	ModeEventual = ConsistencyMode("eventual")
	ModeStrong   = ConsistencyMode("strong")
)

type Peer struct {
	ID      string `json:"id"`
	Address string `json:"address"` 
}

type Config struct {
	ReplicaID       string          `json:"replica_id"`
	Port            int             `json:"port"`
	ConsistencyMode ConsistencyMode `json:"consistency_mode"`
	Peers           []Peer          `json:"peers"`
	NetworkDelayMs int `json:"network_delay_ms"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.ReplicaID == "" {
		return fmt.Errorf("replica_id is required")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if c.ConsistencyMode != ModeEventual && c.ConsistencyMode != ModeStrong {
		return fmt.Errorf("consistency_mode must be 'eventual' or 'strong'")
	}
	return nil
}
