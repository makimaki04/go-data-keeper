package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

func NewLogger(cfgPath string) (*zap.SugaredLogger, error) {
	fmt.Println("loading logger config from: ", cfgPath)

	cfgJSON, err := os.ReadFile(cfgPath)
	if err != nil {
		fmt.Println("couldn't logger config file from: ", cfgPath)
		return nil, err
	}

	var cfg zap.Config

	if err := json.Unmarshal(cfgJSON, &cfg); err != nil {
		fmt.Println("logger config json unmarshal error", err)
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
	}

	sugar := logger.Sugar()

	sugar.Info("logger successfully initialized")
	return sugar, nil
}