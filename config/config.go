package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Gateway struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"gateway"`
	Chains map[string]ChainConfig `yaml:"chains"`
}

// ChainConfig represents configuration for a specific blockchain chain
type ChainConfig struct {
	ChainID         int    `yaml:"chain_id"`
	ContractAddress string `yaml:"contract_address"`
	EventSignature  string `yaml:"event_signature"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// GetGatewayAddr returns the gateway address in the format "host:port"
func (c *Config) GetGatewayAddr() string {
	return fmt.Sprintf("%s:%d", c.Gateway.Address, c.Gateway.Port)
}

// GetChainConfig returns the configuration for a specific chain
func (c *Config) GetChainConfig(chainID string) (*ChainConfig, error) {
	chain, ok := c.Chains[chainID]
	if !ok {
		return nil, fmt.Errorf("chain %s not found in configuration", chainID)
	}
	return &chain, nil
}
