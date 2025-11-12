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

	// 批量获取zones数据
	zoneResponses, err := a.getBatchZoneResponses(ctx, zones)
	if err != nil {
		logger.Errorf("Failed to create batch zone responses: %v", err)
		return nil, err
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

	responses, err := a.getBatchZoneResponses(ctx, []*model.Zone{zone})
	if err != nil {
		return
	}

	resp = responses[0]
	logger.Debugf("Get zone successfully: %+v", resp)
	return
}

func (a *ZoneAdapter) getBatchZoneResponses(ctx context.Context, zones []*model.Zone) ([]*ZoneResponse, error) {
	if len(zones) == 0 {
		return []*ZoneResponse{}, nil
	}

	zoneIDs := make([]int64, len(zones))
	zoneMap := make(map[int64]*model.Zone)

	for i, zone := range zones {
		zoneIDs[i] = zone.ID
		zoneMap[zone.ID] = zone
	}

	hypersByZone, err := a.hyperService.GetHypersByZoneIDs(ctx, zoneIDs)
	if err != nil {
		logger.Errorf("Failed to batch get hypers by zone IDs: %v", err)
		return nil, err
	}

	var hyperIDs []int32
	for _, hypers := range hypersByZone {
		for _, hyper := range hypers {
			hyperIDs = append(hyperIDs, hyper.Hostid)
		}
	}

	instanceCountsByHyper, err := a.instanceService.GetInstanceCountsByHypers(ctx, hyperIDs)
	if err != nil {
		logger.Errorf("Failed to batch get instance counts by hyper IDs: %v", err)
		return nil, err
	}

	responses := make([]*ZoneResponse, len(zones))
	for i, zone := range zones {
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

		hypers := hypersByZone[zone.ID]
		resp.HyperCount = int64(len(hypers))

		for _, hyper := range hypers {
			if hyper.Resource != nil {
				resp.CpuTotal += hyper.Resource.CpuTotal
				resp.Cpu += hyper.Resource.Cpu
				resp.MemoryTotal += hyper.Resource.MemoryTotal
				resp.Memory += hyper.Resource.Memory
				resp.DiskTotal += hyper.Resource.DiskTotal
				resp.Disk += hyper.Resource.Disk
			}

			if count, exists := instanceCountsByHyper[hyper.Hostid]; exists {
				resp.InstanceCount += count
			}
		}

		resp.Memory = resp.Memory / (1024 * 1024)              // Convert KB to GB
		resp.Disk = resp.Disk / (1024 * 1024 * 1024)           // Convert B to GB
		resp.MemoryTotal = resp.MemoryTotal / (1024 * 1024)    // Convert KB to GB
		resp.DiskTotal = resp.DiskTotal / (1024 * 1024 * 1024) // Convert B to GB

		responses[i] = resp
	}

	return responses, nil
}
