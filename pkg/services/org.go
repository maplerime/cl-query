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
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
)

type OrgAdmin struct{}

func (a *OrgAdmin) Get(ctx context.Context, id int64) (org *model.Organization, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid org ID: %d", id)
		logger.Error("%v", err)
		return
	}
	db := DB()
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	org = &model.Organization{Model: model.Model{ID: id}}
	err = db.Where(where).Take(org).Error
	if err != nil {
		logger.Error("Failed to query user, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, org.Owner)
	if !permit {
		logger.Error("Not authorized to read the org")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *OrgAdmin) GetOrgByUUID(ctx context.Context, uuID string) (org *model.Organization, err error) {
	db := DB()
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	org = &model.Organization{}
	err = db.Where(where).Where("uuid = ?", uuID).Take(org).Error
	if err != nil {
		logger.Error("Failed to query org, %v", err)
		return
	}
	permit := memberShip.ValidateOwner(model.Reader, org.Owner)
	if !permit {
		logger.Error("Not authorized to read the org")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *OrgAdmin) GetOrgByName(name string) (org *model.Organization, err error) {
	org = &model.Organization{}
	db := DB()
	err = db.Take(org, &model.Organization{Name: name}).Error
	return
}

func (a *OrgAdmin) GetOrgName(ctx context.Context, id int64) (name string) {
	org := &model.Organization{Model: model.Model{ID: id}}
	ctx, db := GetContextDB(ctx)
	err := db.Take(org, &model.Organization{Name: name}).Error
	if err != nil {
		logger.Error("DB failed to query org", err)
		return
	}
	name = org.Name
	return
}
