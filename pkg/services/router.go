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

type StaticRoute struct {
	Destination string `json:"destination"`
	Nexthop     string `json:"nexthop"`
}

type SubnetIface struct {
	Address string         `json:"ip_address"`
	MacAddr string         `json:"mac_address"`
	Vni     int64          `json:"vni"`
	Routes  []*StaticRoute `json:"routes,omitempty"`
}

type RouterAdmin struct{}

func (a *RouterAdmin) Get(ctx context.Context, id int64) (router *model.Router, err error) {
	if id <= 0 {
		logger.Error("returning nil router")
		return
	}
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	router = &model.Router{Model: model.Model{ID: id}}
	if err = db.Preload("Subnets").Where(where).Take(router).Error; err != nil {
		logger.Error("Failed to query router", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, router.Owner)
	if !permit {
		logger.Error("Not authorized to read the router")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *RouterAdmin) GetRouterByUUID(ctx context.Context, uuID string) (router *model.Router, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	router = &model.Router{}
	err = db.Preload("Subnets").Where(where).Where("uuid = ?", uuID).Take(router).Error
	if err != nil {
		logger.Error("Failed to query router, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, router.Owner)
	if !permit {
		logger.Error("Not authorized to read the router")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *RouterAdmin) GetRouterByName(ctx context.Context, name string) (router *model.Router, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	router = &model.Router{}
	err = db.Preload("Subnets").Where(where).Where("name = ?", name).Take(router).Error
	if err != nil {
		logger.Error("Failed to query router, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, router.Owner)
	if !permit {
		logger.Error("Not authorized to read the router")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *RouterAdmin) GetRouter(ctx context.Context, reference *BaseReference) (router *model.Router, err error) {
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = fmt.Errorf("Router base reference must be provided with either uuid or name")
		return
	}
	if reference.ID != "" {
		router, err = a.GetRouterByUUID(ctx, reference.ID)
		return
	}
	if reference.Name != "" {
		router, err = a.GetRouterByName(ctx, reference.Name)
		return
	}
	return
}

func (a *RouterAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, routers []*model.Router, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	where := memberShip.GetWhere()
	routers = []*model.Router{}
	if err = db.Model(&model.Router{}).Where(where).Where(query).Count(&total).Error; err != nil {
		logger.Error("DB failed to count router, %v", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Subnets").Where(where).Where(query).Find(&routers).Error; err != nil {
		logger.Error("DB failed to query routers, %v", err)
		return
	}
	permit := memberShip.CheckPermission(model.Admin)
	if permit {
		db = db.Offset(0).Limit(-1)
		for _, router := range routers {
			router.OwnerInfo = &model.Organization{Model: model.Model{ID: router.Owner}}
			if err = db.Take(router.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				return
			}
		}
	}
	return
}

func (a *SubnetAdmin) AddressStatistics(ctx context.Context, subnet *model.Subnet) (total, allocated, reserved, idle int64, err error) {
	db := DB()
	query := db.Model(&model.Address{}).
		Select(`
			COUNT(*) as total,
			SUM(CASE WHEN allocated = 't' THEN 1 ELSE 0 END) as allocated,
			SUM(CASE WHEN reserved = 't' THEN 1 ELSE 0 END) as reserved,
			SUM(CASE WHEN allocated = 'f' AND reserved = 'f' THEN 1 ELSE 0 END) as idle
		`).
		Where("subnet_id = ? AND address != ?", subnet.ID, subnet.Gateway)

	var result struct {
		Total     int64
		Allocated int64
		Reserved  int64
		Idle      int64
	}

	if err = query.Scan(&result).Error; err != nil {
		logger.Error("Failed to count addresses for subnet", err)
		return
	}

	return result.Total, result.Allocated, result.Reserved, result.Idle, nil
}
