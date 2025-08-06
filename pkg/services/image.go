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

type ImageAdmin struct{}

func (a *ImageAdmin) GetImageByUUID(ctx context.Context, uuID string) (image *model.Image, err error) {
	ctx, db := GetContextDB(ctx)
	image = &model.Image{}
	err = db.Where("uuid = ?", uuID).Take(image).Error
	if err != nil {
		logger.Error("Failed to query image, %v", err)
		return
	}
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized to get image")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *ImageAdmin) GetImageByName(ctx context.Context, name string) (image *model.Image, err error) {
	ctx, db := GetContextDB(ctx)
	image = &model.Image{}
	err = db.Where("name = ?", name).Take(image).Error
	if err != nil {
		logger.Error("Failed to query image, %v", err)
		return
	}
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized to get image")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *ImageAdmin) Get(ctx context.Context, id int64) (image *model.Image, err error) {
	if id <= 0 {
		err = fmt.Errorf("Invalid image ID: %d", id)
		logger.Error(err)
		return
	}
	ctx, db := GetContextDB(ctx)
	image = &model.Image{Model: model.Model{ID: id}}
	err = db.Take(image).Error
	if err != nil {
		logger.Error("DB failed to query image, %v", err)
		return
	}
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized to get image")
		err = fmt.Errorf("Not authorized")
		return
	}
	return
}

func (a *ImageAdmin) GetImage(ctx context.Context, reference *BaseReference) (image *model.Image, err error) {
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = fmt.Errorf("Image base reference must be provided with either uuid or name")
		return
	}
	if reference.ID != "" {
		image, err = a.GetImageByUUID(ctx, reference.ID)
		return
	}
	if reference.Name != "" {
		image, err = a.GetImageByName(ctx, reference.Name)
		return
	}
	return
}

func (a *ImageAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, images []*model.Image, err error) {
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	images = []*model.Image{}
	if err = db.Model(&model.Image{}).Where(query).Count(&total).Error; err != nil {
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Where(query).Find(&images).Error; err != nil {
		return
	}

	return
}
