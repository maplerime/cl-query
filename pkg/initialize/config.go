/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2024 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Account management service configuration
 *
**/
package initialize

import (
	"context"

	"github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/conf"
)

var (
	// LeaveOnInt quit server on int signal
	LeaveOnInt = true
	// LeaveOnTerm quit server on terminate signal
	LeaveOnTerm = true

	ctx *context.Context
)

// loadServiceCfg returns the configurations for this service
func LoadServiceCfg(c *context.Context, programName string) {
	ctx = c
	cfg := &common.ServiceConfig{}

	myViper, err := conf.New(programName)
	if err != nil {
		logger.Panic("Failed to initialize configuration: ", err)
	}
	err = myViper.Unmarshal(cfg)
	if err != nil {
		logger.Panic("Error loading configuration: ", err)
	}

	logger.Infof("Initialize configuration done: %+v", cfg)

	common.Config = cfg
}
