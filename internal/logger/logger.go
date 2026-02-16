// Package logger provides zap logger initialization from configuration.
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// NewLogger builds a sugared zap logger from the JSON configuration at cfgPath.
// NewLogger returns an error if the configuration can't be read, decoded, or used to build a logger.
func NewLogger(cfgPath string) (*zap.SugaredLogger, error) {
	cfgJSON, err := os.ReadFile(cfgPath)
	if err != nil {
		fmt.Println("couldn't read logger config from: ", cfgPath)
		return nil, err
	}

	var cfg zap.Config

	if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
		fmt.Println("logger config json unmarshal error", err)
		return nil, err
	}

	for _, path := range cfg.OutputPaths {
		if path == "stdout" || path == "stderr" {
			continue
		}

		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Println("logger mkfir error: ", err)
			return nil, err
		}
	}

	logger, err := cfg.Build()
	if err != nil {
		fmt.Println("logger build error: ", err)
		return nil, err
	}

	sugar := logger.Sugar()

	sugar.Info("logger successfully init")
	return sugar, nil
}
