/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Hyper resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"web/src/model"

	"github.com/gin-gonic/gin"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

type HyperResponse struct {
	*BaseReference
	Cpu    int64 `json:"cpu"`
	Memory int64 `json:"memory"`
	Disk   int64 `json:"disk"`
	Hostid int32 `json:"hostid"`
}

type HyperListResponse struct {
	*ResourceReference
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Hypers []*HyperResponse `json:"hypers"`
}

type HyperFilters struct {
	Hostname string `json:"hostname,omitempty"`
	Status   *int   `json:"status,omitempty" binding:"omitempty"`
}

type HyperAdapter struct {
	BaseAdapter
	service *services.HyperAdmin
}

func NewHyperAdapter() *HyperAdapter {
	logger.Debug("Creating new Hyper adapter")
	return &HyperAdapter{
		service: &services.HyperAdmin{},
	}
}

func (a *HyperAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Hyper filters: %+v", filtersMap)

	filters, err := ParseFilters[HyperFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// hostname查询
	if filters.Hostname != "" {
		conditions = append(conditions, fmt.Sprintf("hostname like '%%%s%%'", filters.Hostname))
		logger.Debugf("Added hostname filter: %s", filters.Hostname)
	}

	// 状态查询
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = %d", *filters.Status))
		logger.Debugf("Added status filter: %d", *filters.Status)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *HyperAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Hyper list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	// 调用 service 层
	total, hypers, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Hypers (total: %d)", len(hypers), total)

	// 构建响应
	hyperResponses := make([]*HyperResponse, len(hypers))
	for i, hyper := range hypers {
		hyperResp, err := a.getHyperResponse(ctx, hyper)
		if err != nil {
			logger.Errorf("Failed to create hyper response: %v", err)
			return nil, err
		}
		hyperResponses[i] = hyperResp
	}

	// 返回响应
	hyperListResp := &HyperListResponse{
		ResourceReference: &ResourceReference{
			ID:   "hyper-list",
			Name: "Hyper List",
		},
		Total:  int(total),
		Limit:  req.Limit,
		Hypers: hyperResponses,
	}

	logger.Debugf("List hypers successfully: %+v", hyperListResp)
	return hyperListResp, nil
}

func (a *HyperAdapter) Get(c *gin.Context, id string) (interface{}, error) {
	return nil, nil
}

func (a *HyperAdapter) getHyperResponse(ctx context.Context, hyper *model.Hyper) (*HyperResponse, error) {
	hyperResp := &HyperResponse{
		BaseReference: &BaseReference{
			ID:   strconv.Itoa(int(hyper.Hostid)),
			Name: hyper.Hostname,
		},
		Cpu:    hyper.Resource.Cpu,
		Memory: hyper.Resource.Memory,
		Disk:   hyper.Resource.Disk,
		Hostid: hyper.Hostid,
	}
	return hyperResp, nil
}
