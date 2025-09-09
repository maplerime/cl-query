/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

var (
	subnetAdmin = &SubnetAdmin{}
	vniMax      = 16777215
	vniMin      = 4096
)

type SubnetAdmin struct{}

func init() {
	rand.Seed(time.Now().UnixNano())
	return
}

func getValidVni(ctx context.Context) (vni int, err error) {
	ctx, db := GetContextDB(ctx)
	count := 1
	for count > 0 {
		vni = rand.Intn(vniMax-vniMin) + vniMin
		if err = db.Model(&model.Subnet{}).Where("vlan = ?", vni).Count(&count).Error; err != nil {
			logger.Error("Failed to query existing vlan, %v", err)
			return
		}
	}
	return
}

func (a *SubnetAdmin) Get(ctx context.Context, id int64) (subnet *model.Subnet, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid subnet ID: %d", id)
		logger.Error(err)
		return
	}
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	subnet = &model.Subnet{Model: model.Model{ID: id}}
	err = db.Preload("Router").Preload("Group").Take(subnet).Error
	if err != nil {
		logger.Error("DB failed to query subnet ", err)
		return
	}
	if subnet.RouterID > 0 {
		subnet.Router = &model.Router{Model: model.Model{ID: subnet.RouterID}}
		err = db.Take(subnet.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			return
		}
	}
	if subnet.Type == "internal" {
		permit := memberShip.ValidateOwner(model.Reader, subnet.Owner)
		if !permit {
			logger.Error("Not authorized to read the subnet")
			err = fmt.Errorf("Not authorized")
			return
		}
	}
	return
}

func (a *SubnetAdmin) GetSubnetByUUID(ctx context.Context, uuID string) (subnet *model.Subnet, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	subnet = &model.Subnet{}
	err = db.Preload("Router").Preload("Group").Where("uuid = ?", uuID).Take(subnet).Error
	if err != nil {
		logger.Error("Failed to query subnet, %v", err)
		return
	}
	if subnet.RouterID > 0 {
		subnet.Router = &model.Router{Model: model.Model{ID: subnet.RouterID}}
		err = db.Take(subnet.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			return
		}
	}
	if subnet.Type == "internal" {
		permit := memberShip.ValidateOwner(model.Reader, subnet.Owner)
		if !permit {
			logger.Error("Not authorized to read the subnet")
			err = fmt.Errorf("Not authorized")
			return
		}
	}
	return
}

func (a *SubnetAdmin) GetSubnetByName(ctx context.Context, name string) (subnet *model.Subnet, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	subnet = &model.Subnet{}
	err = db.Preload("Router").Preload("Group").Where("name = ?", name).Take(subnet).Error
	if err != nil {
		logger.Error("Failed to query subnet ", err)
		return
	}
	if subnet.RouterID > 0 {
		subnet.Router = &model.Router{Model: model.Model{ID: subnet.RouterID}}
		err = db.Take(subnet.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			return
		}
	}
	if subnet.Type == "internal" {
		permit := memberShip.ValidateOwner(model.Reader, subnet.Owner)
		if !permit {
			logger.Error("Not authorized to read the subnet")
			err = fmt.Errorf("Not authorized")
			return
		}
	}
	return
}

func (a *SubnetAdmin) GetSubnet(ctx context.Context, reference *BaseReference) (subnet *model.Subnet, err error) {
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = fmt.Errorf("Subnet base reference must be provided with either uuid or name")
		return
	}
	if reference.ID != "" {
		subnet, err = a.GetSubnetByUUID(ctx, reference.ID)
		return
	}
	if reference.Name != "" {
		subnet, err = a.GetSubnetByName(ctx, reference.Name)
		return
	}
	return
}

func (a *SubnetAdmin) CountIdleAddressesForSubnet(ctx context.Context, subnet *model.Subnet) (int64, error) {
	ctx, db := GetContextDB(ctx)
	var idleCount int64

	err := db.Model(&model.Address{}).
		Where("subnet_id = ?", subnet.ID).
		Where("allocated = ?", "f").
		Where("address != ?", subnet.Gateway).
		Count(&idleCount).Error

	if err != nil {
		if err.Error() != "record not found" {
			return 0, fmt.Errorf("failed to count idle addresses for subnet %s: %v", subnet.UUID, err)
		}
	}

	return idleCount, nil
}

func (a *SubnetAdmin) List(ctx context.Context, offset, limit int64, order, query string, hasIdleIP bool) (total int64, subnets []*model.Subnet, err error) {
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	m := GetMemberShip(ctx)
	where := ""
	if m.OrgName == "admin" && m.Role == model.Admin {
		where = ""
	} else {
		where = fmt.Sprintf("subnets.owner = %d", m.OrgID)
	}
	subnets = []*model.Subnet{}

	// 始终连接 addresses 表
	baseQuery := db.
		Model(&model.Subnet{}).
		Joins("LEFT JOIN addresses ON subnets.id = addresses.subnet_id").
		Where(where).
		Where(query)

	if hasIdleIP {
		baseQuery = baseQuery.
			Where("addresses.allocated = ? AND addresses.reserved = ? AND addresses.address != subnets.gateway",
				false,
				false,
			)
	}

	baseQuery = baseQuery.Group("subnets.id")

	// 计算总数
	if err = baseQuery.Count(&total).Error; err != nil {
		return
	}

	// 查询结果
	resultQuery := baseQuery.Select("subnets.*").Offset(offset).Limit(limit)
	resultQuery = dbs.Sortby(resultQuery, order)
	if err = resultQuery.Preload("Group").Preload("Router").Find(&subnets).Error; err != nil {
		return
	}

	permit := m.CheckPermission(model.Admin)
	if permit {
		ownerQuery := db.Offset(0).Limit(-1)
		for _, subnet := range subnets {
			subnet.OwnerInfo = &model.Organization{Model: model.Model{ID: subnet.Owner}}
			if err = ownerQuery.Take(subnet.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				return
			}
		}
	}

	return
}

func (a *SubnetAdmin) AddressList(ctx context.Context, offset, limit int64, order, query string) (total int64, addresses []*model.Address, err error) {
	db := DB()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "address::inet"
	}

	addresses = []*model.Address{}
	if err = db.Model(&model.Address{}).Where(query).Count(&total).Error; err != nil {
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Subnet").Where(query).Find(&addresses).Error; err != nil {
		return
	}
	return
}

func (a *SubnetAdmin) GetAddressByUUID(ctx context.Context, uuID string) (address *model.Address, err error) {
	ctx, db := GetContextDB(ctx)
	address = &model.Address{}
	err = db.Preload("Subnet").Where("uuid = ?", uuID).Take(address).Error
	if err != nil {
		logger.Error("Failed to query address, %v", err)
		return
	}
	return
}

func (a *SubnetAdmin) GetAddressesBySubnet(ctx context.Context, subnetID int64) (addresses []*model.Address, err error) {
	ctx, db := GetContextDB(ctx)
	addresses = []*model.Address{}
	err = db.Where("subnet_id = ?", subnetID).Order("address::inet").Find(&addresses).Error
	if err != nil {
		logger.Error("Failed to query addresses by subnet_id, %v", err)
		return
	}
	return
}
