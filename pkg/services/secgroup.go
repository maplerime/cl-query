/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package services

import (
	"context"
	"web/src/common"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

type SecgroupAdmin struct{}

func (a *SecgroupAdmin) Get(ctx context.Context, id int64) (secgroup *model.SecurityGroup, err error) {
	if id <= 0 {
		return a.GetSecgroupByName(ctx, common.SystemDefaultSGName)
	}
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()
	secgroup = &model.SecurityGroup{Model: model.Model{ID: id}}
	err = db.Where(where).Take(secgroup).Error
	if err != nil {
		logger.Error("DB failed to query secgroup ", err)
		err = NewCLError(ErrSecurityGroupNotFound, "Failed to find security group", err)
		return
	}
	if secgroup.RouterID > 0 {
		secgroup.Router = &model.Router{Model: model.Model{ID: secgroup.RouterID}}
		err = db.Take(secgroup.Router).Error
		if err != nil {
			logger.Error("DB failed to qeury router", err)
			err = NewCLError(ErrRouterNotFound, "Failed to find router", err)
			return
		}
	}
	if secgroup.Name != "system-default" {
		permit := memberShip.ValidateOwner(model.Reader, secgroup.Owner)
		if !permit {
			logger.Error("Not authorized to get security group")
			err = NewCLError(ErrPermissionDenied, "Not authorized to get security group", nil)
			return
		}
	}
	return
}

func (a *SecgroupAdmin) GetSecgroupByUUID(ctx context.Context, uuID string) (secgroup *model.SecurityGroup, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	secgroup = &model.SecurityGroup{}
	err = db.Where(where).Where("uuid = ?", uuID).Take(secgroup).Error
	if err != nil {
		logger.Error("Failed to query secgroup ", err)
		err = NewCLError(ErrSecurityGroupNotFound, "Failed to find security group", err)
		return
	}
	if secgroup.RouterID > 0 {
		secgroup.Router = &model.Router{Model: model.Model{ID: secgroup.RouterID}}
		err = db.Take(secgroup.Router).Error
		if err != nil {
			logger.Error("DB failed to qeury router", err)
			err = NewCLError(ErrRouterNotFound, "Failed to find router", err)
			return
		}
	}
	if secgroup.Name != "system-default" {
		permit := memberShip.ValidateOwner(model.Reader, secgroup.Owner)
		if !permit {
			logger.Error("Not authorized to get security group")
			err = NewCLError(ErrPermissionDenied, "Not authorized to get security group", nil)
			return
		}
	}
	return
}

func (a *SecgroupAdmin) GetSecgroupByName(ctx context.Context, name string) (secgroup *model.SecurityGroup, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	secgroup = &model.SecurityGroup{}
	err = db.Where("name = ?", name).Take(secgroup).Error
	if err != nil {
		logger.Error("Failed to query secgroup ", err)
		err = NewCLError(ErrSecurityGroupNotFound, "Failed to find security group", err)
		return
	}
	if secgroup.RouterID > 0 {
		secgroup.Router = &model.Router{Model: model.Model{ID: secgroup.RouterID}}
		err = db.Take(secgroup.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			err = NewCLError(ErrRouterNotFound, "Failed to find router", err)
			return
		}
	}
	if secgroup.Name != "system-default" {
		permit := memberShip.ValidateOwner(model.Reader, secgroup.Owner)
		if !permit {
			logger.Error("Not authorized to get security group")
			err = NewCLError(ErrPermissionDenied, "Not authorized to get security group", nil)
			return
		}
	}
	return
}

func (a *SecgroupAdmin) GetSecurityGroup(ctx context.Context, reference *BaseReference) (secgroup *model.SecurityGroup, err error) {
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = NewCLError(ErrInvalidParameter, "Security group base reference must be provided with either uuid or name", nil)
		return
	}
	if reference.ID != "" {
		secgroup, err = a.GetSecgroupByUUID(ctx, reference.ID)
		return
	}
	if reference.Name != "" {
		secgroup, err = a.GetSecgroupByName(ctx, reference.Name)
		return
	}
	return
}

func (a *SecgroupAdmin) GetSecgroupInterfaces(ctx context.Context, secgroup *model.SecurityGroup) (err error) {
	ctx, db := GetContextDB(ctx)
	err = db.Model(secgroup).Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses").Preload("SecondAddresses.Subnet").Preload("SiteSubnets").Where("instance > 0").Related(&secgroup.Interfaces, "Interfaces").Error
	if err != nil {
		logger.Error("Failed to query secgroup, %v", err)
		err = NewCLError(ErrSecurityGroupNotFound, "Failed to find security group", err)
		return
	}
	return
}

func (a *SecgroupAdmin) GetInterfaceSecgroups(ctx context.Context, iface *model.Interface) (err error) {
	ctx, db := GetContextDB(ctx)
	err = db.Model(iface).Related(&iface.SecurityGroups, "Security_Groups").Error
	if err != nil {
		logger.Error("Failed to query interface, %v", err)
		err = NewCLError(ErrInterfaceNotFound, "Failed to find interface", err)
		return
	}
	return
}

func (a *SecgroupAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, secgroups []*model.SecurityGroup, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		return
	}
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}
	logger.Debugf("The query in admin console is %s", query)

	where := memberShip.GetWhere()
	secgroups = []*model.SecurityGroup{}
	if err = db.Model(&model.SecurityGroup{}).Where(where).Where(query).Count(&total).Error; err != nil {
		logger.Error("DB failed to count security group(s), %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count security group(s)", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Where(where).Where(query).Find(&secgroups).Error; err != nil {
		logger.Error("DB failed to query security group(s), %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to query security group(s)", err)
		return
	}
	for _, secgroup := range secgroups {
		if secgroup.RouterID > 0 {
			secgroup.Router = &model.Router{Model: model.Model{ID: secgroup.RouterID}}
			err = db.Take(secgroup.Router).Error
			if err != nil {
				logger.Error("DB failed to qeury router", err)
				err = nil
				continue
			}
		}
	}
	permit = memberShip.CheckPermission(model.Admin)
	if permit {
		db = db.Offset(0).Limit(-1)
		for _, sg := range secgroups {
			sg.OwnerInfo = &model.Organization{Model: model.Model{ID: sg.Owner}}
			if err = db.Take(sg.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				err = nil
				continue
			}
		}
	}

	return
}
