package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type TsConfig struct {
	CompilerOptions CompilerOptions `json:"compilerOptions"`
	Files           []string        `json:"files"`
	Include         []string        `json:"include"`
	Exclude         []string        `json:"exclude"`
}

type CompilerOptions struct {
	Target           string `json:"target"`
	Module           string `json:"module"`
	ModuleResolution string `json:"moduleResolution"`
}

// ParseTsConfig reads and parses a tsconfig.json file
func ParseTsConfig(path string) (*TsConfig, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tsconfig: %w", err)
	}

	var config TsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse tsconfig: %w", err)
	}

	return &config, nil
}

// GetProjectRoot returns the directory containing the tsconfig
func (c *TsConfig) GetProjectRoot(tsconfigPath string) string {
	return filepath.Dir(tsconfigPath)
}
