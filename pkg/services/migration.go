/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"context"
	"fmt"
	"web/src/model"

	"github.com/maplerime/cl-query/pkg/dbs"

	. "github.com/maplerime/cl-query/pkg/common"
)

type MigrationAdmin struct{}

func (a *MigrationAdmin) GetMigrationByUUID(ctx context.Context, uuID string) (migration *model.Migration, err error) {
	ctx, db := GetContextDB(ctx)
	migration = &model.Migration{}
	err = db.Preload("Instance").Preload("Phases").Where("uuid = ?", uuID).Take(migration).Error
	if err != nil {
		logger.Error("Failed to query migration, %v", err)
		err = NewCLError(ErrMigrationNotFound, "Failed to find migration", err)
		return
	}
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Admin)
	if !permit {
		logger.Error("Not authorized to get migration")
		err = NewCLError(ErrPermissionDenied, "Not authorized to get migration", nil)
		return
	}
	return
}

func (a *MigrationAdmin) GetMigrationByName(ctx context.Context, name string) (migration *model.Migration, err error) {
	ctx, db := GetContextDB(ctx)
	migration = &model.Migration{}
	err = db.Where("name = ?", name).Take(migration).Error
	if err != nil {
		logger.Error("Failed to query migration, %v", err)
		err = NewCLError(ErrMigrationNotFound, "Failed to find migration", err)
		return
	}
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Admin)
	if !permit {
		logger.Error("Not authorized to get migration")
		err = NewCLError(ErrPermissionDenied, "Not authorized to get migration", nil)
		return
	}
	return
}

func (a *MigrationAdmin) Get(ctx context.Context, id int64) (migration *model.Migration, err error) {
	if id <= 0 {
		err = NewCLError(ErrInvalidParameter, fmt.Sprintf("Invalid migration ID: %d", id), nil)
		logger.Error(err)
		return
	}
	ctx, db := GetContextDB(ctx)
	migration = &model.Migration{Model: model.Model{ID: id}}
	err = db.Take(migration).Error
	if err != nil {
		logger.Error("DB failed to query migration, %v", err)
		err = NewCLError(ErrMigrationNotFound, "Failed to find migration", err)
		return
	}
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Admin)
	if !permit {
		logger.Error("Not authorized to get migration")
		err = NewCLError(ErrPermissionDenied, "Not authorized to get migration", nil)
		return
	}
	return
}

func (a *MigrationAdmin) GetMigration(ctx context.Context, reference *BaseReference) (migration *model.Migration, err error) {
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = NewCLError(ErrInvalidParameter, "Migration base reference must be provided with either uuid or name", nil)
		return
	}
	if reference.ID != "" {
		migration, err = a.GetMigrationByUUID(ctx, reference.ID)
		return
	}
	if reference.Name != "" {
		migration, err = a.GetMigrationByName(ctx, reference.Name)
		return
	}
	return
}

func (a *MigrationAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, migrations []*model.Migration, err error) {
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	if query != "" {
		query = fmt.Sprintf("name like '%%%s%%'", query)
	}
	migrations = []*model.Migration{}
	if err = db.Model(&model.Migration{}).Where(query).Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to count migrations", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Instance").Preload("Phases").Where(query).Find(&migrations).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to query migrations", err)
		return
	}

	return
}
