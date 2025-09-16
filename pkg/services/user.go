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
		err = NewCLError(ErrInvalidParameter, "Invalid user ID", nil)
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
		err = NewCLError(ErrUserNotFound, "Failed to query user", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, user.Owner)
	if !permit {
		logger.Error("Not authorized to read the user")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the user", nil)
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
		err = NewCLError(ErrUserNotFound, "User not found", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, user.Owner)
	if !permit {
		logger.Error("Not authorized to read the user")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the user", nil)
		return
	}
	return
}

func (a *UserAdmin) GetUserByName(name string) (user *model.User, err error) {
	db := DB()
	user = &model.User{}
	if err = db.Where("username = ?", name).Take(user).Error; err != nil {
		logger.Error("DB failed to get user", err)
		err = NewCLError(ErrUserNotFound, "User not found", err)
		return
	}
	return
}
