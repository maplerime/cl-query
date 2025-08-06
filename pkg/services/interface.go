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

	"github.com/jinzhu/gorm"
	. "github.com/maplerime/cl-query/pkg/common"
)

type InterfaceInfo struct {
	PublicIps      []*model.FloatingIp
	Subnets        []*model.Subnet
	MacAddress     string
	IpAddress      string
	Count          int
	SiteSubnets    []*model.Subnet
	Inbound        int32
	Outbound       int32
	AllowSpoofing  bool
	SecurityGroups []*model.SecurityGroup
}

type InterfaceAdmin struct{}

func (a *InterfaceAdmin) Get(ctx context.Context, id int64) (iface *model.Interface, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid interface ID: %d", id)
		logger.Debug(err)
		return
	}
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	iface = &model.Interface{Model: model.Model{ID: id}}
	err = db.Preload("SiteSubnets").Preload("SecurityGroups").Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("addresses.updated_at")
	}).Preload("SecondAddresses.Subnet").Take(iface).Error
	if err != nil {
		logger.Debug("DB failed to query interface, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, iface.Owner)
	if !permit {
		logger.Debug("Not authorized to read the subnet")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *InterfaceAdmin) GetInterfaceByUUID(ctx context.Context, uuID string) (iface *model.Interface, err error) {
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	ctx, db := GetContextDB(ctx)
	iface = &model.Interface{}
	err = db.Preload("SiteSubnets").Preload("SecurityGroups").Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("addresses.updated_at")
	}).Preload("SecondAddresses.Subnet").Where(where).Where("uuid = ?", uuID).Take(iface).Error
	if err != nil {
		logger.Debug("DB failed to query interface, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, iface.Owner)
	if !permit {
		logger.Debug("Not authorized to read the subnet")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *InterfaceAdmin) List(ctx context.Context, offset, limit int64, order string, instance *model.Instance) (total int64, interfaces []*model.Interface, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.ValidateOwner(model.Reader, instance.Owner)
	if !permit {
		logger.Debug("Not authorized for this operation")
		err = fmt.Errorf("Not authorized")
		return
	}
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	where := fmt.Sprintf("instance = %d", instance.ID)
	wm := memberShip.GetWhere()
	if wm != "" {
		where = fmt.Sprintf("%s and %s", where, wm)
	}
	interfaces = []*model.Interface{}
	if err = db.Model(&model.Interface{}).Where(where).Count(&total).Error; err != nil {
		logger.Debug("DB failed to count security rule(s), %v", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("SiteSubnets").Preload("SecurityGroups").Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("addresses.updated_at")
	}).Preload("SecondAddresses.Subnet").Where(where).Find(&interfaces).Error; err != nil {
		logger.Debug("DB failed to query security rule(s), %v", err)
		return
	}

	return
}
