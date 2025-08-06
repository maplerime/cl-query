/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: base model
 *
**/

package model

import (
	"github.com/maplerime/cl-query/pkg/dbs"
	"github.com/maplerime/cl-query/utils/logging"
)

var logger = logging.MustGetLogger("model")

func InitModel() {
	logger.Debugf("Initializing models ...")
	dbs.AutoMigrate()
}
