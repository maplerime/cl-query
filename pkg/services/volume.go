/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package services

import (
	"context"
	"fmt"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

type VolumeAdmin struct{}

func (a *VolumeAdmin) Get(ctx context.Context, id int64) (volume *model.Volume, err error) {
	if id <= 0 {
		err = NewCLError(ErrInvalidParameter, fmt.Sprintf("Invalid volume ID: %d", id), nil)
		logger.Error(err)
		return
	}
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	volume = &model.Volume{Model: model.Model{ID: id}}
	if err = db.Preload("Instance").Where(where).Take(volume).Error; err != nil {
		logger.Error("Failed to query volume, %v", err)
		err = NewCLError(ErrVolumeNotFound, "Failed to query volume", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, volume.Owner)
	if !permit {
		logger.Error("Not authorized to read the volume")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the volume", nil)
		return
	}
	return
}

func (a *VolumeAdmin) GetVolumeByUUID(ctx context.Context, uuID string) (volume *model.Volume, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	volume = &model.Volume{}
	where := memberShip.GetWhere()
	err = db.Preload("Instance").Where(where).Where("uuid = ?", uuID).Take(volume).Error
	if err != nil {
		logger.Error("DB: query volume failed", err)
		err = NewCLError(ErrVolumeNotFound, "Volume not found", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, volume.Owner)
	if !permit {
		logger.Error("Not authorized to read the volume")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the volume", nil)
		return
	}
	return
}

func (a *VolumeAdmin) ListVolume(ctx context.Context, offset, limit int64, order, query string) (total int64, volumes []*model.Volume, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	where := memberShip.GetWhere()

	volumes = []*model.Volume{}
	if err = db.Model(&model.Volume{}).Where(where).Where(query).Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to count volumes", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Instance").Where(where).Where(query).Find(&volumes).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to query volumes", err)
		return
	}
	permit := memberShip.CheckPermission(model.Admin)
	if permit {
		db = db.Offset(0).Limit(-1)
		for _, vol := range volumes {
			vol.OwnerInfo = &model.Organization{Model: model.Model{ID: vol.Owner}}
			if err = db.Take(vol.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				err = NewCLError(ErrOwnerNotFound, "Owner organization not found", err)
				return
			}
		}
	}

	return
}

func (a *VolumeAdmin) Count(ctx context.Context) (count int64, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	bootingWhere := fmt.Sprintf("booting=%t", false)
	if err = db.Model(&model.Volume{}).Where(where).Where(bootingWhere).Count(&count).Error; err != nil {
		logger.Error("Failed to count volumes", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count volumes", err)
	}
	return
}

func (a *VolumeAdmin) SumSize(ctx context.Context) (total int64, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	var result struct {
		Total int64
	}
	if err = db.Model(&model.Volume{}).Select("SUM(size) as total").Where(where).Scan(&result).Error; err != nil {
		logger.Error("Failed to sum volume size", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to sum volume size", err)
		return
	}
	return result.Total, nil
}
