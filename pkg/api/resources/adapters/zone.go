/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Zone resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	. "github.com/maplerime/cl-query/pkg/common"
	"strconv"
	"strings"
	"web/src/model"

	"github.com/gin-gonic/gin"

	"github.com/maplerime/cl-query/pkg/services"
)

type ZoneResponse struct {
	*ResourceReference
	Remark        string `json:"remark"`
	Default       bool   `json:"default"`
	CpuTotal      int64  `json:"cpu_total"`
	Cpu           int64  `json:"cpu"`
	MemoryTotal   int64  `json:"memory_total"`
	Memory        int64  `json:"memory"`
	DiskTotal     int64  `json:"disk_total"`
	Disk          int64  `json:"disk"`
	HyperCount    int64  `json:"hyper_count"`
	InstanceCount int64  `json:"instance_count"`
}

type ZoneListResponse struct {
	Total int             `json:"total"`
	Limit int             `json:"limit"`
	Zones []*ZoneResponse `json:"zones"`
}

type ZoneFilters struct {
	Name string `json:"name,omitempty"`
}

type ZoneAdapter struct {
	BaseAdapter
	service         *services.ZoneAdmin
	hyperService    *services.HyperAdmin
	instanceService *services.InstanceAdmin
}

func NewZoneAdapter() *ZoneAdapter {
	logger.Debug("Creating new Zone adapter")
	return &ZoneAdapter{
		service:         &services.ZoneAdmin{},
		hyperService:    &services.HyperAdmin{},
		instanceService: &services.InstanceAdmin{},
	}
}

func (a *ZoneAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Zone filters: %+v", filtersMap)

	filters, err := ParseFilters[ZoneFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// name查询
	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name like '%%%s%%'", filters.Name))
		logger.Debugf("Added name filter: %s", filters.Name)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *ZoneAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Zone list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	// 调用 service 层
	total, zones, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Zones (total: %d)", len(zones), total)

	// 构建响应
	zoneResponses := make([]*ZoneResponse, len(zones))
	for i, zone := range zones {
		zoneResp, err := a.getZoneResponse(ctx, zone)
		if err != nil {
			logger.Errorf("Failed to create zone response: %v", err)
			return nil, err
		}
		zoneResponses[i] = zoneResp
	}

	// 返回响应
	zoneListResp := &ZoneListResponse{
		Total: int(total),
		Limit: req.Limit,
		Zones: zoneResponses,
	}

	logger.Debugf("List zones successfully: %+v", zoneListResp)
	return zoneListResp, nil
}

func (a *ZoneAdapter) Get(c *gin.Context, zoneName string) (resp interface{}, err error) {
	logger.Debugf("Starting Zone get query with name: %s", zoneName)

	ctx := c.Request.Context()
	zone, err := a.service.GetZoneByName(ctx, zoneName)
	if err != nil {
		return
	}

	resp, err = a.getZoneResponse(ctx, zone)
	if err != nil {
		return
	}

	logger.Debugf("Get zone successfully: %+v", resp)
	return
}

func (a *ZoneAdapter) getZoneResponse(ctx context.Context, zone *model.Zone) (*ZoneResponse, error) {
	resp := &ZoneResponse{
		ResourceReference: &ResourceReference{
			ID:        strconv.FormatInt(zone.ID, 10),
			Name:      zone.Name,
			CreatedAt: zone.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: zone.UpdatedAt.Format(TimeStringForMat),
		},
		Default: zone.Default,
		Remark:  zone.Remark,
	}

	// 获取该zone下的所有hyper并汇总资源
	err := a.aggregateHyperResources(ctx, zone.ID, resp)
	if err != nil {
		logger.Errorf("Failed to aggregate hyper resources for zone %d: %v", zone.ID, err)
		return nil, err
	}

	return resp, nil
}

func (a *ZoneAdapter) aggregateHyperResources(ctx context.Context, zoneID int64, resp *ZoneResponse) error {
	// 获取该zone下的所有hyper来汇总资源
	query := fmt.Sprintf("zone_id = %d", zoneID)
	_, hypers, err := a.hyperService.List(ctx, 0, -1, "", query)
	if err != nil {
		return err
	}

	// hyper总数
	resp.HyperCount = int64(len(hypers))

	// 汇总资源
	for _, hyper := range hypers {
		if hyper.Resource != nil {
			// CPU
			resp.CpuTotal += hyper.Resource.CpuTotal
			resp.Cpu += hyper.Resource.Cpu

			// Memory
			resp.MemoryTotal += hyper.Resource.MemoryTotal
			resp.Memory += hyper.Resource.Memory

			// Disk
			resp.DiskTotal += hyper.Resource.DiskTotal
			resp.Disk += hyper.Resource.Disk
		}
	}

	// 单位转换
	resp.Memory = resp.Memory / (1024 * 1024)              // Convert KB to GB
	resp.Disk = resp.Disk / (1024 * 1024 * 1024)           // Convert B to GB
	resp.MemoryTotal = resp.MemoryTotal / (1024 * 1024)    // Convert KB to GB
	resp.DiskTotal = resp.DiskTotal / (1024 * 1024 * 1024) // Convert B to GB

	// 统计该zone下所有hyper的VM数量
	vmCount, err := a.instanceService.GetInstanceCountByZone(ctx, zoneID)
	if err != nil {
		logger.Errorf("Failed to count VMs for zone %d: %v", zoneID, err)
	} else {
		resp.InstanceCount = vmCount
	}

	return nil
}
