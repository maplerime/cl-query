/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

History:
   Date     Who ID    Description
   -------- --- ---   -----------
   01/13/19 nanjj  Initial code

*/

package services

import (
	"context"
	"fmt"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/utils/logging"
)

var (
	userAdmin = &UserAdmin{}
)

var logger = logging.MustGetLogger("services")

type UserAdmin struct{}

func (a *UserAdmin) Get(ctx context.Context, id int64) (user *model.User, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid user ID: %d", id)
		logger.Error("%v", err)
		return
	}
	db := DB()
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	user = &model.User{Model: model.Model{ID: id}}
	err = db.Where(where).Take(user).Error
	if err != nil {
		logger.Error("Failed to query user, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, user.Owner)
	if !permit {
		logger.Error("Not authorized to read the user")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *UserAdmin) GetUserByUUID(ctx context.Context, uuID string) (user *model.User, err error) {
	db := DB()
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	user = &model.User{}
	err = db.Where(where).Where("uuid = ?", uuID).Take(user).Error
	if err != nil {
		logger.Error("Failed to query user, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, user.Owner)
	if !permit {
		logger.Error("Not authorized to read the user")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *UserAdmin) GetUserByName(name string) (user *model.User, err error) {
	db := DB()
	user = &model.User{}
	if err = db.Where("username = ?", name).Take(user).Error; err != nil {
		logger.Error("DB failed to get user", err)
		return
	}
	return
}
