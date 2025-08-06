/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2024 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Query service
 *
**/

package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/initialize"
	"github.com/maplerime/cl-query/utils/logging"
)

// gracefulTimeout controls how long we wait before forcefully terminating
// const gracefulTimeout = 5 * time.Second

var (
	logger   = logging.MustGetLogger("query/main")
	svc      *ApiSVC
	metadata = common.GetSvcMetadata()
)

// ApiSVC ...
type ApiSVC struct {
	sysContext context.Context
}

func init() {
	if svc == nil {
		svc = &ApiSVC{}
	}
	err := svc.init()
	if err != nil {
		panic(err.Error())
	}
}

func StartUserAPI() error {
	return svc.startUserAPI()
}

func (svc *ApiSVC) init() (err error) {
	logger.Debugf("initializing %s service ...", common.ProgramName)
	// load configuration
	svc.sysContext = context.Background()
	initialize.LoadServiceCfg(&svc.sysContext, common.ProgramName)
	// init logger
	loggerCfg := common.Config.RPCSvc.HTTPSvcCfg.Log
	logging.InitRollingBackend(loggerCfg.LogFile, loggerCfg.MaxSize, loggerCfg.MaxBackups, loggerCfg.MaxAge)
	logging.InitFromSpec(loggerCfg.LogLevel)
	// set service context
	metadata.SVCID = common.Config.RPCSvc.SVCID

	// 1. init casbin enforcer ...
	//logger.Info("Initializing casbin enforcer ...")
	//initialize.InitCasbinEnforcer(common.Config.IAM.CasbinConf)
	//logger.Info("Casbin enforcer initialized successfully")
	// 2. init gin
	logger.Info("Initializing gin ...")
	if common.Config.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	logger.Info("Gin initialized successfully")

	// 3. Initialize the database connection
	logger.Info("Initializing database connection ...")
	initialize.DBInit()
	logger.Info("Database connection initialized successfully")

	return nil
}

func (svc *ApiSVC) startUserAPI() error {
	logger.Debugf("starting user api service ...")
	listenAddr := fmt.Sprintf("%s:%d", common.Config.RPCSvc.HTTPSvcCfg.ListenAddress, common.Config.RPCSvc.HTTPSvcCfg.ListenPort)
	logger.Infof("Starting user api service on %s ...", listenAddr)

	router := NewRouter(&svc.sysContext)
	if err := router.Run(listenAddr); err != nil {
		logger.Errorf("Failed to start user api service, %+v", err)
		return err
	}
	logger.Infof("Started user api service")
	return nil
}
