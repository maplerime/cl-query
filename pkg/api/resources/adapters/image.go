/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Image resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	"strings"
	"web/src/model"

	"github.com/gin-gonic/gin"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

type ImageResponse struct {
	*ResourceReference
	OSCode       string `json:"os_code"`
	Size         int64  `json:"size"`
	Format       string `json:"format"`
	Architecture string `json:"architecture"`
	User         string `json:"user"`
	Status       string `json:"status"`
	OSVersion    string `json:"os_version"`
	BootLoader   string `json:"boot_loader"`
	OsFamily     string `json:"os_family"`
	IsRescue     bool   `json:"is_rescue"`
}

type ImageListResponse struct {
	Offset int              `json:"offset"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Images []*ImageResponse `json:"images"`
}

type ImageFilters struct {
	UUIDs  []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	Name   string   `json:"name,omitempty"`
	Status string   `json:"status,omitempty" binding:"omitempty"`
}

type ImageAdapter struct {
	BaseAdapter
	service *services.ImageAdmin
}

func NewImageAdapter() *ImageAdapter {
	logger.Debug("Creating new Image adapter")
	return &ImageAdapter{
		service: &services.ImageAdmin{},
	}
}

func (a *ImageAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Image filters: %+v", filtersMap)

	filters, err := ParseFilters[ImageFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// name查询
	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name like '%%%s%%'", filters.Name))
		logger.Debugf("Added name filter: %s", filters.Name)
	}

	// UUIDs查询
	if filters.UUIDs != nil {
		if len(filters.UUIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
			logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
		} else {
			conditions = append(conditions, "1=0")
		}
	}

	// 状态查询
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = '%s'", filters.Status))
		logger.Debugf("Added status filter: %s", filters.Status)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *ImageAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Image list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	// 调用 service 层
	total, images, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Images (total: %d)", len(images), total)

	// 返回响应
	imageListResp := &ImageListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(images),
	}

	// 构建响应
	imageListResp.Images = make([]*ImageResponse, imageListResp.Limit)
	for i, image := range images {
		imageListResp.Images[i], err = a.getImageResponse(ctx, image)
		if err != nil {
			logger.Errorf("Failed to create image response: %v", err)
			return nil, err
		}
	}

	logger.Debugf("List images successfully: %+v", imageListResp)
	return imageListResp, nil
}

func (a *ImageAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting Image get query with ID: %s", id)

	ctx := c.Request.Context()
	image, err := a.service.GetImageByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getImageResponse(ctx, image)
	if err != nil {
		return
	}

	logger.Debugf("Get image successfully: %+v", resp)
	return
}

func (a *ImageAdapter) getImageResponse(ctx context.Context, image *model.Image) (*ImageResponse, error) {
	imageResp := &ImageResponse{
		ResourceReference: &ResourceReference{
			ID:        image.UUID,
			Name:      image.Name,
			CreatedAt: image.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: image.UpdatedAt.Format(TimeStringForMat),
		},
		OSCode:       image.OSCode,
		Size:         image.Size,
		Format:       image.Format,
		Architecture: image.Architecture,
		User:         image.UserName,
		Status:       image.Status,
		OSVersion:    image.OsVersion,
		BootLoader:   image.BootLoader,
		OsFamily:     image.OsFamily,
		IsRescue:     image.IsRescue,
	}
	return imageResp, nil
}
