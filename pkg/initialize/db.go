/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

History:
   Date     Who ID    Description
   -------- --- ---   -----------
   01/13/19 nanjj  Initial code

*/

package initialize

import (
	"github.com/jinzhu/gorm"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
	"github.com/maplerime/cl-query/pkg/model"
)

func DBInit() {
	model.InitModel()
	dbs.InitDB(&Config.Database)
	db := dbs.DB()
	logger.Debugf("Database initialized: %+v", db)
	dbs.AutoUpgrade("01-admin-upgrade", func(db *gorm.DB) (err error) {
		return
	})
}
