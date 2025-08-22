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

type FloatingIps struct {
	Instance  int64  `json:"instance"`
	PublicIp  string `json:"public_ip"`
	PrivateIp string `json:"private_ip"`
}

type FloatingIpAdmin struct{}

func (a *FloatingIpAdmin) Get(ctx context.Context, id int64) (floatingIp *model.FloatingIp, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid floatingIp ID: %d", id)
		logger.Error(err)
		return
	}
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()
	floatingIp = &model.FloatingIp{Model: model.Model{ID: id}}
	err = db.Preload("Interface").Preload("Interface.SecurityGroups").Preload("Interface.Address").Preload("Interface.Address.Subnet").Preload("Subnet").Preload("Subnet.Group").Preload("Group").Where(where).Take(floatingIp).Error
	if err != nil {
		logger.Error("DB failed to query floatingIp ", err)
		return
	}
	if floatingIp.InstanceID > 0 {
		floatingIp.Instance = &model.Instance{Model: model.Model{ID: floatingIp.InstanceID}}
		err = db.Take(floatingIp.Instance).Error
		if err != nil {
			logger.Error("DB failed to query instance ", err)
			return
		}
		instance := floatingIp.Instance
		err = db.Preload("Address").Preload("Address.Subnet").Where("instance = ? and primary_if = true", instance.ID).Find(&instance.Interfaces).Error
		if err != nil {
			logger.Error("Failed to query interfaces %v", err)
			return
		}
	}
	if floatingIp.RouterID > 0 {
		floatingIp.Router = &model.Router{Model: model.Model{ID: floatingIp.RouterID}}
		err = db.Take(floatingIp.Router).Error
		if err != nil {
			logger.Error("DB failed to query instance ", err)
			return
		}
	}

	return
}

func (a *FloatingIpAdmin) GetFloatingIpByUUID(ctx context.Context, uuID string) (floatingIp *model.FloatingIp, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	floatingIp = &model.FloatingIp{}
	err = db.Preload("Interface").Preload("Interface.SecurityGroups").Preload("Interface.Address").Preload("Interface.Address.Subnet").Preload("Subnet").Preload("Subnet.Group").Preload("Group").Where(where).Where("uuid = ?", uuID).Take(floatingIp).Error
	if err != nil {
		logger.Error("Failed to query floatingIp, %v", err)
		return
	}
	if floatingIp.InstanceID > 0 {
		floatingIp.Instance = &model.Instance{Model: model.Model{ID: floatingIp.InstanceID}}
		err = db.Take(floatingIp.Instance).Error
		if err != nil {
			logger.Error("DB failed to query instance ", err)
			return
		}
		instance := floatingIp.Instance
		err = db.Preload("Address").Preload("Address.Subnet").Where("instance = ? and primary_if = true", instance.ID).Find(&instance.Interfaces).Error
		if err != nil {
			logger.Error("Failed to query interfaces %v", err)
			return
		}
	}
	if floatingIp.RouterID > 0 {
		floatingIp.Router = &model.Router{Model: model.Model{ID: floatingIp.RouterID}}
		err = db.Take(floatingIp.Router).Error
		if err != nil {
			logger.Error("DB failed to query instance ", err)
			return
		}
	}

	return
}

func (a *FloatingIpAdmin) List(ctx context.Context, offset, limit int64, order, query string, intQuery string) (total int64, floatingIps []*model.FloatingIp, err error) {
	memberShip := GetMemberShip(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()
	floatingIps = []*model.FloatingIp{}
	if err = db.Model(&model.FloatingIp{}).Where(where).Where(query).Where(intQuery).Count(&total).Error; err != nil {
		logger.Error("DB failed to count floating ip(s), %v", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Group").Preload("Instance").Preload("Instance.Zone").Preload("Interface").Preload("Interface.Address").Preload("Interface.Address.Subnet").Preload("Subnet").Preload("Subnet.Group").Where(where).Where(query).Where(intQuery).Find(&floatingIps).Error; err != nil {
		logger.Error("DB failed to query floating ip(s), %v", err)
		return
	}
	for _, fip := range floatingIps {
		if fip.InstanceID > 0 {
			if fip.Instance != nil && fip.Instance.ID > 0 {
				instance := fip.Instance
				err = db.Preload("Address").Where("instance = ? and primary_if = true", instance.ID).Find(&instance.Interfaces).Error
				if err != nil {
					logger.Error("Failed to query interfaces ", err)
					err = nil
					continue
				}
			} else {
				fip.Instance = &model.Instance{Model: model.Model{ID: fip.InstanceID}}
				err = db.Take(fip.Instance).Error
				if err != nil {
					logger.Error("DB failed to query instance ", err)
					err = nil
					continue
				}
				instance := fip.Instance
				err = db.Preload("Address").Where("instance = ? and primary_if = true", instance.ID).Find(&instance.Interfaces).Error
				if err != nil {
					logger.Error("Failed to query interfaces ", err)
					err = nil
					continue
				}
			}
		}

		if fip.RouterID > 0 {
			fip.Router = &model.Router{Model: model.Model{ID: fip.RouterID}}
			err = db.Take(fip.Router).Error
			if err != nil {
				logger.Error("DB failed to query router ", err)
				err = nil
				continue
			}
		}
	}
	permit := memberShip.CheckPermission(model.Admin)
	if permit {
		db = db.Offset(0).Limit(-1)
		for _, fip := range floatingIps {
			fip.OwnerInfo = &model.Organization{Model: model.Model{ID: fip.Owner}}
			if err = db.Take(fip.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				return
			}
		}
	}

	return
}

func (a *FloatingIpAdmin) GetFloatingIpByAddress(ctx context.Context, ipAddress string) (floatingIp *model.FloatingIp, err error) {
	db := DB()
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	floatingIp = &model.FloatingIp{}
	err = db.Where(where).Where("fip_address = ? OR int_address = ?", ipAddress, ipAddress).Take(floatingIp).Error
	if err != nil {
		logger.Error("Failed to query floatingIp, %v", err)
		return
	}
	if floatingIp.InstanceID > 0 {
		floatingIp.Instance = &model.Instance{Model: model.Model{ID: floatingIp.InstanceID}}
		err = db.Take(floatingIp.Instance).Error
		if err != nil {
			logger.Error("DB failed to query instance ", err)
			return
		}
	}
	permit := memberShip.CheckPermission(model.Admin)
	if permit {
		floatingIp.OwnerInfo = &model.Organization{Model: model.Model{ID: floatingIp.Owner}}
		if err = db.Take(floatingIp.OwnerInfo).Error; err != nil {
			logger.Error("Failed to query owner info", err)
			return
		}
	}
	return
}
