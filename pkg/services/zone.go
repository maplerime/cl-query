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

type ZoneAdmin struct{}

func (a *ZoneAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, zones []*model.Zone, err error) {
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "hostid"
	}
	if query != "" {
		query = fmt.Sprintf("name like '%%%s%%'", query)
	}

	zones = []*model.Zone{}
	if err = db.Model(&model.Zone{}).Where(query).Count(&total).Error; err != nil {
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Zone").Where("hostid >= 0").Where(query).Find(&zones).Error; err != nil {
		return
	}
	db = db.Offset(0).Limit(-1)
	return
}

func (a *ZoneAdmin) Get(ctx context.Context, id int64) (zone *model.Zone, err error) {
	ctx, db := GetContextDB(ctx)
	zone = &model.Zone{ID: id}
	if err = db.Take(zone).Error; err != nil {
		logger.Error("Failed to query zone", err)
		return
	}
	return
}

func (a *ZoneAdmin) GetZoneByName(ctx context.Context, name string) (zone *model.Zone, err error) {
	ctx, db := GetContextDB(ctx)
	zone = &model.Zone{}
	err = db.Where("name = ?", name).Take(zone).Error
	if err != nil {
		logger.Error("Failed to query zone, %v", err)
		return
	}
	return
}
