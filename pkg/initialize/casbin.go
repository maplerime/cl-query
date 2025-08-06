/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: casbin enforcer initialization
 *
**/

package initialize

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"

	"github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/utils/logging"
)

var logger = logging.MustGetLogger("initialize")

func InitCasbinEnforcer(conf string) {
	logger.Debugf("Initializing casbin enforcer with conf: %s", conf)
	enforcer, err := newEnforcer(conf)
	if err != nil {
		logger.Fatalf("Failed to create casbin enforcer: %v", err)
	}
	common.CasbinEnforcer = enforcer
	logger.Debugf("Casbin enforcer initialized successfully")
}

func newEnforcer(conf string) (enforcer *casbin.Enforcer, err error) {
	logger.Debugf("Creating casbin enforcer with conf: %s", conf)
	adapter, err := gormadapter.NewAdapter("", "")
	/*
		adapter, err := gormadapter.NewAdapterByDBUseTableName(

			"",
			"casbin_rule",
		)
	*/
	if err != nil {
		return nil, err
	}
	casbinModel := model.NewModel()
	err = casbinModel.LoadModelFromText(conf)
	if err != nil {
		return nil, err
	}
	enforcer, err = casbin.NewEnforcer(casbinModel, adapter)
	if err != nil {
		return nil, err
	}

	err = enforcer.LoadPolicy()
	if err != nil {
		return
	}
	logger.Debugf("Casbin enforcer created successfully")
	return
}
