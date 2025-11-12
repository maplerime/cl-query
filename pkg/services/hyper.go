/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"context"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

type HyperAdmin struct{}

func (a *HyperAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, hypers []*model.Hyper, err error) {
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "hostid"
	}

	hypers = []*model.Hyper{}
	if err = db.Model(&model.Hyper{}).Where("hostid >= 0").Where(query).Count(&total).Error; err != nil {
		return 0, nil, NewCLError(ErrSQLSyntaxError, "Failed to count hypervisors", err)
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Zone").Where("hostid >= 0").Where(query).Find(&hypers).Error; err != nil {
		return 0, nil, NewCLError(ErrSQLSyntaxError, "Failed to retrieve hypervisors", err)
	}
	db = db.Offset(0).Limit(-1)
	for _, hyper := range hypers {
		hyper.Resource = &model.Resource{}
		err = db.Where("hostid = ?", hyper.Hostid).Take(hyper.Resource).Error
	}

	return
}

func (a *HyperAdmin) GetHyperByHostid(ctx context.Context, hostid int32) (hyper *model.Hyper, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Admin)
	if !permit {
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		logger.Error("Not authorized for this operation", err)
		return
	}
	_, db := GetContextDB(ctx)
	hyper = &model.Hyper{}
	if err = db.Preload("Zone").Where("hostid = ?", hostid).Take(hyper).Error; err != nil {
		logger.Error("Failed to query hypervisor", err)
		return nil, NewCLError(ErrHypervisorNotFound, "Specified hypervisor not found", err)
	}

	// Load resource information
	hyper.Resource = &model.Resource{}
	err = db.Where("hostid = ?", hyper.Hostid).Take(hyper.Resource).Error
	if err != nil {
		logger.Warning("Failed to query hypervisor resource, setting defaults", err)
		hyper.Resource = &model.Resource{
			Hostid: hyper.Hostid,
		}
	}
	return
}

func (a *HyperAdmin) GetHypersByHostids(ctx context.Context, hostids []int32) (hypers map[int32]*model.Hyper, err error) {
	if len(hostids) == 0 {
		return make(map[int32]*model.Hyper), nil
	}

	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Admin)
	if !permit {
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		logger.Error("Not authorized for this operation", err)
		return
	}

	_, db := GetContextDB(ctx)

	var hyperList []*model.Hyper
	if err = db.Preload("Zone").Where("hostid IN (?)", hostids).Find(&hyperList).Error; err != nil {
		logger.Error("Failed to batch query hypervisors", err)
		return nil, NewCLError(ErrHypervisorNotFound, "Specified hypervisor not found", err)
	}

	hypers = make(map[int32]*model.Hyper)
	for _, hyper := range hyperList {
		hypers[hyper.Hostid] = hyper
	}

	return
}

func (a *HyperAdmin) GetHyperByHostname(ctx context.Context, hostname string) (hyper *model.Hyper, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Admin)
	if !permit {
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		logger.Error("Not authorized for this operation", err)
		return
	}
	ctx, db := GetContextDB(ctx)
	hyper = &model.Hyper{}
	if err = db.Where("hostname = ?", hostname).Take(hyper).Error; err != nil {
		logger.Error("Failed to query hypervisor", err)
		return nil, NewCLError(ErrHypervisorNotFound, "Specified hypervisor not found", err)
	}
	return
}

func (a *HyperAdmin) GetHypersByZoneIDs(ctx context.Context, zoneIDs []int64) (hypersByZone map[int64][]*model.Hyper, err error) {
	if len(zoneIDs) == 0 {
		return make(map[int64][]*model.Hyper), nil
	}

	_, db := GetContextDB(ctx)

	var hypers []*model.Hyper
	if err = db.Where("hostid >= 0 AND zone_id IN (?)", zoneIDs).Find(&hypers).Error; err != nil {
		logger.Errorf("Failed to batch query hypervisors by zone IDs: %v", err)
		return nil, NewCLError(ErrSQLSyntaxError, "Failed to retrieve hypervisors", err)
	}

	var hostids []int32
	hyperMap := make(map[int32]*model.Hyper)
	for _, hyper := range hypers {
		hostids = append(hostids, hyper.Hostid)
		hyperMap[hyper.Hostid] = hyper
	}

	var resources []*model.Resource
	if len(hostids) > 0 {
		if err = db.Where("hostid IN (?)", hostids).Find(&resources).Error; err != nil {
			logger.Errorf("Failed to batch query resources: %v", err)
			return nil, NewCLError(ErrSQLSyntaxError, "Failed to retrieve resources", err)
		}

		for _, resource := range resources {
			if hyper, exists := hyperMap[resource.Hostid]; exists {
				hyper.Resource = resource
			}
		}
	}

	hypersByZone = make(map[int64][]*model.Hyper)
	for _, hyper := range hypers {
		if hyper.Resource == nil {
			hyper.Resource = &model.Resource{Hostid: hyper.Hostid}
		}
		if hypersByZone[hyper.ZoneID] == nil {
			hypersByZone[hyper.ZoneID] = []*model.Hyper{}
		}
		hypersByZone[hyper.ZoneID] = append(hypersByZone[hyper.ZoneID], hyper)
	}

	return hypersByZone, nil
}
