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
		err = fmt.Errorf("Invalid subnet ID: %d", id)
		logger.Error(err)
		return
	}
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	volume = &model.Volume{Model: model.Model{ID: id}}
	if err = db.Preload("Instance").Where(where).Take(volume).Error; err != nil {
		logger.Error("Failed to query volume, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, volume.Owner)
	if !permit {
		logger.Error("Not authorized to read the volume")
		err = fmt.Errorf("Not authorized")
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
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, volume.Owner)
	if !permit {
		logger.Error("Not authorized to read the volume")
		err = fmt.Errorf("Not authorized")
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
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Instance").Where(where).Where(query).Find(&volumes).Error; err != nil {
		return
	}
	permit := memberShip.CheckPermission(model.Admin)
	if permit {
		db = db.Offset(0).Limit(-1)
		for _, vol := range volumes {
			vol.OwnerInfo = &model.Organization{Model: model.Model{ID: vol.Owner}}
			if err = db.Take(vol.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				return
			}
		}
	}

	return
}
