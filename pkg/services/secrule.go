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

type SecruleAdmin struct{}

func (a *SecruleAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, secrules []*model.SecurityRule, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
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

	where := memberShip.GetWhere()
	secrules = []*model.SecurityRule{}
	if err = db.Model(&model.SecurityRule{}).Where(where).Where(query).Count(&total).Error; err != nil {
		logger.Error("DB failed to count security rule(s), %v", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Where(where).Where(query).Find(&secrules).Error; err != nil {
		logger.Error("DB failed to query security rule(s), %v", err)
		return
	}

	return
}

func (a *SecruleAdmin) Get(ctx context.Context, id int64, secgroup *model.SecurityGroup) (secrule *model.SecurityRule, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid security rule ID: %d", id)
		logger.Error(err)
		return
	}
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	db := DB()
	secrule = &model.SecurityRule{Model: model.Model{ID: id}}
	err = db.Where(where).Take(secrule).Error
	if err != nil {
		logger.Error("Failed to query secrule", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, secrule.Owner)
	if !permit {
		logger.Error("Not authorized to get security group")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *SecruleAdmin) GetSecruleByUUID(ctx context.Context, uuID string) (secrule *model.SecurityRule, err error) {
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	db := DB()
	secrule = &model.SecurityRule{}
	err = db.Where(where).Where("uuid = ?", uuID).Take(secrule).Error
	if err != nil {
		logger.Error("Failed to query secrule", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, secrule.Owner)
	if !permit {
		logger.Error("Not authorized to get security group")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}
