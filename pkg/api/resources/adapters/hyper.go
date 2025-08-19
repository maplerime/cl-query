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

	"github.com/maplerime/cl-query/pkg/services"
)

type HyperResponse struct {
	Hostid       int32   `json:"hostid"`
	Hostname     string  `json:"hostname"`
	Status       int32   `json:"status"`
	StatusName   string  `json:"status_name"`
	Parentid     int32   `json:"parentid"`
	Children     int32   `json:"children"`
	HostIP       string  `json:"host_ip"`
	RouteIP      string  `json:"route_ip"`
	VirtType     string  `json:"virt_type"`
	CpuOverRate  float32 `json:"cpu_over_rate"`
	MemOverRate  float32 `json:"mem_over_rate"`
	DiskOverRate float32 `json:"disk_over_rate"`
	ZoneID       int64   `json:"zone_id"`
	ZoneName     string  `json:"zone_name"`
	Remark       string  `json:"remark"`
	Cpu          int64   `json:"cpu"`
	Memory       int64   `json:"memory"`
	Disk         int64   `json:"disk"`
	CpuTotal     int64   `json:"cpu_total"`
	MemoryTotal  int64   `json:"memory_total"`
	DiskTotal    int64   `json:"disk_total"`
	VMTotal      int64   `json:"vm_total"`
}

type HyperListResponse struct {
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
	service         *services.HyperAdmin
	instanceService *services.InstanceAdmin
}

func NewHyperAdapter() *HyperAdapter {
	logger.Debug("Creating new Hyper adapter")
	return &HyperAdapter{
		service:         &services.HyperAdmin{},
		instanceService: &services.InstanceAdmin{},
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
		hyperResp := a.getHyperResponse(ctx, hyper)
		if err != nil {
			logger.Errorf("Failed to create hyper response: %v", err)
			return nil, err
		}
		hyperResponses[i] = hyperResp
	}

	// 返回响应
	hyperListResp := &HyperListResponse{
		Total:  int(total),
		Limit:  req.Limit,
		Hypers: hyperResponses,
	}

	logger.Debugf("List hypers successfully: %+v", hyperListResp)
	return hyperListResp, nil
}

func (a *HyperAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting Hyper get query with HostID: %s", id)

	hostID, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		return
	}

	ctx := c.Request.Context()
	hyper, err := a.service.GetHyperByHostid(ctx, int32(hostID))
	if err != nil {
		return
	}

	resp = a.getHyperResponse(ctx, hyper)
	logger.Debugf("Get hyper successfully: %+v", resp)
	return
}

func (a *HyperAdapter) getHyperResponse(ctx context.Context, hyper *model.Hyper) *HyperResponse {
	resp := &HyperResponse{
		Hostid:       hyper.Hostid,
		Hostname:     hyper.Hostname,
		Status:       hyper.Status,
		StatusName:   hyper.GetStatus(),
		Parentid:     hyper.Parentid,
		Children:     hyper.Children,
		HostIP:       hyper.HostIP,
		RouteIP:      hyper.RouteIP,
		VirtType:     hyper.VirtType,
		CpuOverRate:  hyper.CpuOverRate,
		MemOverRate:  hyper.MemOverRate,
		DiskOverRate: hyper.DiskOverRate,
		ZoneID:       hyper.ZoneID,
		Remark:       hyper.Remark,
	}

	if hyper.Zone != nil {
		resp.ZoneName = hyper.Zone.Name
	}

	if hyper.Resource != nil {
		resp.Cpu = hyper.Resource.Cpu
		resp.Memory = hyper.Resource.Memory / 1024             // Convert KB to MB
		resp.Disk = hyper.Resource.Disk / (1024 * 1024 * 1024) // Convert B to GB
		resp.CpuTotal = hyper.Resource.CpuTotal
		resp.MemoryTotal = hyper.Resource.MemoryTotal / 1024             // Convert KB to MB
		resp.DiskTotal = hyper.Resource.DiskTotal / (1024 * 1024 * 1024) // Convert B to GB
	}

	// VM Total
	count, err := a.instanceService.GetInstanceCountByHyper(ctx, hyper.Hostid)
	if err != nil {
		logger.Errorf("Failed to get instance count for hyper %d: %v", hyper.Hostid, err)
	} else {
		resp.VMTotal = count
	}

	return resp
}
