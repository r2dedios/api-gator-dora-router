package config

import (
	ag "exate-dora-router/internal/apigator"
	"fmt"
	"go.uber.org/zap"
	ini "gopkg.in/ini.v1"
	"net/http"
	"strings"
	"time"
)

const (
	iniAPIGatorPrefix = "api_gator"
	iniRouterSection  = "router"
	iniCommonSection  = "common"
)

func LoadConfig(fileName string, logger *zap.Logger) (*ag.APIGatorRouter, error) {
	cfg, err := ini.Load(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var commonConfig ag.APIGatorConfig
	if err := cfg.Section(iniCommonSection).MapTo(&commonConfig); err != nil {
		return nil, fmt.Errorf("failed to parse common config: %v", err)
	}

	var APIGators []*ag.APIGatorTarget
	for _, section := range cfg.Sections() {
		if strings.HasPrefix(section.Name(), iniAPIGatorPrefix) {
			var target ag.APIGatorTarget
			if err := section.MapTo(&target); err != nil {
				return nil, fmt.Errorf("failed to parse API Gator config: %v", err)
			}
			target.Config = &commonConfig
			target.Client = &http.Client{
				Timeout: commonConfig.Timeout * time.Second,
			}
			target.Logger = logger
			APIGators = append(APIGators, &target)
		}
	}

	var router ag.APIGatorRouter
	if err := cfg.Section(iniRouterSection).MapTo(&router); err != nil {
		return nil, fmt.Errorf("failed to parse APIGatorRouter config: %v", err)
	}
	router.APIGatorTargets = APIGators

	logger.Info("Configuration Loaded Successfully", zap.Int("apigators_count", len(router.APIGatorTargets)))

	return &router, nil
}
