/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"fmt"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

type ImageStorageAdmin struct{}

func (a *ImageStorageAdmin) List(offset, limit int64, order string, image *model.Image, query string) (total int64, storages []*model.ImageStorage, err error) {
	db := DB()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	if query != "" {
		query = fmt.Sprintf("pool_id = '%s'", query)
	}

	storages = []*model.ImageStorage{}
	if err = db.Model(&model.ImageStorage{}).Where("image_id = ?", image.ID).Where(query).Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Find to count image storage", nil)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Where("image_id = ?", image.ID).Where(query).Find(&storages).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Find to list image storage", nil)
		return
	}

	return
}
