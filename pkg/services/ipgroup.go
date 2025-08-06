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
	"strings"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

type IpGroupAdmin struct{}

func (a *IpGroupAdmin) Get(ctx context.Context, id int64) (ipGroup *model.IpGroup, err error) {
	logger.Debugf("Enter IpGroupAdmin.Get, id=%d", id)
	if id <= 0 {
		err = fmt.Errorf("Invalid ipGroup ID: %d", id)
		logger.Errorf("%v", err)
		return
	}
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	ipGroup = &model.IpGroup{Model: model.Model{ID: id}}
	err = db.Where(where).Preload("Subnets").Preload("DictionaryType").Take(ipGroup).Error
	if err != nil {
		logger.Errorf("Failed to query ipGroup, %v", err)
		return
	}
	logger.Debugf("IpGroupAdmin.Get: success, ipGroup=%+v", ipGroup)
	return
}

func (a *IpGroupAdmin) GetIpGroupByUUID(ctx context.Context, uuID string) (ipGroup *model.IpGroup, err error) {
	logger.Debugf("Enter IpGroupAdmin.GetIpGroupByUUID, uuID=%s", uuID)
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	ipGroup = &model.IpGroup{}
	err = db.Where(where).Where("uuid = ?", uuID).Preload("Subnets").Preload("FloatingIPs").Preload("FloatingIPs.Subnet").Preload("DictionaryType").Take(ipGroup).Error
	if err != nil {
		logger.Errorf("Failed to query ipGroup, %v", err)
		return
	}
	logger.Debugf("IpGroupAdmin.GetIpGroupByUUID: success, uuid=%s, ipGroup=%+v", ipGroup.UUID, ipGroup)
	return
}

func (a *IpGroupAdmin) GetIpGroupByName(ctx context.Context, name string) (ipGroup *model.IpGroup, err error) {
	logger.Debugf("Enter IpGroupAdmin.GetIpGroupByName, name=%s", name)
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	ipGroup = &model.IpGroup{}
	err = db.Where(where).Where("name = ?", name).Preload("Subnets").Preload("DictionaryType").Take(ipGroup).Error
	if err != nil {
		logger.Errorf("Failed to query ipGroup, %v", err)
		return
	}
	logger.Debugf("IpGroupAdmin.GetIpGroupByName: success, name=%s, ipGroup=%+v", name, ipGroup)
	return
}

func (a *IpGroupAdmin) GetIpGroup(ctx context.Context, reference *BaseReference) (ipGroup *model.IpGroup, err error) {
	logger.Debugf("Enter IpGroupAdmin.GetIpGroup, reference=%+v", reference)
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = fmt.Errorf("IpGroup base reference must be provided with either uuid or name")
		logger.Errorf("Exit IpGroupAdmin.GetIpGroup with error")
		return
	}
	if reference.ID != "" {
		ipGroup, err = a.GetIpGroupByUUID(ctx, reference.ID)
		logger.Debugf("Exit IpGroupAdmin.GetIpGroup by UUID, uuid=%s, ipGroup=%+v, err=%v", reference.ID, ipGroup, err)
		return
	}
	logger.Debugf("Exit IpGroupAdmin.GetIpGroup with nil result")
	return
}

func (a *IpGroupAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, ipGroups []*model.IpGroup, err error) {
	logger.Debugf("Enter IpGroupAdmin.List, offset=%d, limit=%d, order=%s, query=%s", offset, limit, order, query)
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	if limit == 0 {
		limit = 16
	}
	if order == "" {
		order = "created_at"
	}
	ipGroups = []*model.IpGroup{}
	if err = db.Model(&model.IpGroup{}).Where(where).Where(query).Count(&total).Error; err != nil {
		logger.Errorf("IpGroupAdmin.List: count error, err=%v", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Subnets").Preload("DictionaryType").Preload("FloatingIPs").Preload("FloatingIPs.Subnet").Where(where).Where(query).Find(&ipGroups).Error; err != nil {
		logger.Errorf("IpGroupAdmin.List: find error, err=%v", err)
		return
	}
	for _, ipGroup := range ipGroups {
		var names []string
		for _, subnet := range ipGroup.Subnets {
			names = append(names, subnet.Name)
		}
		ipGroup.SubnetNames = strings.Join(names, ",")
		var floatingIpNames []string
		for _, floatingIp := range ipGroup.FloatingIPs {
			floatingIpNames = append(floatingIpNames, floatingIp.Name)
		}
		ipGroup.FloatingIPNames = strings.Join(floatingIpNames, ",")
	}
	logger.Debugf("IpGroupAdmin.List: success, total=%d, count=%d", total, len(ipGroups))
	return
}
