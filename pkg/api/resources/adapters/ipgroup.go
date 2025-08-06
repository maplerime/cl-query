/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: IpGroup resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	"strings"
	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"

	"github.com/gin-gonic/gin"

	"github.com/maplerime/cl-query/pkg/services"
)

type FloatingIpWithInfo struct {
	*BaseReference
	Vlan       int64  `json:"vlan"`
	IPAddress  string `json:"ip_address"`
	FipAddress string `json:"fip_address"`
	Type       string `json:"type"`
}

type IpGroupResponse struct {
	*ResourceReference
	Type           string                `json:"type"`
	Dictionary     *BaseReference        `json:"dictionaries,omitempty"`
	SubnetNames    string                `json:"subnet_names"`
	Subnets        []*BaseReference      `json:"subnets,omitempty"`
	FloatingIps    []*FloatingIpWithInfo `json:"floating_ips,omitempty"`
	IPCount        int64                 `json:"ip_count"`
	IdleCount      int64                 `json:"idle_count"`
	ReservedCount  int64                 `json:"reserved_count"`
	AllocatedCount int64                 `json:"allocated_count"`
}

type IpGroupListResponse struct {
	Offset   int                `json:"offset"`
	Total    int                `json:"total"`
	Limit    int                `json:"limit"`
	IpGroups []*IpGroupResponse `json:"ipgroups"`
}

type IpGroupFilters struct {
	UUIDs []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	Name  string   `json:"name,omitempty"`
}

type IpGroupAdapter struct {
	BaseAdapter
	service       *services.IpGroupAdmin
	subnetService *services.SubnetAdmin
}

func NewIpGroupAdapter() *IpGroupAdapter {
	logger.Debug("Creating new IpGroup adapter")
	return &IpGroupAdapter{
		service:       &services.IpGroupAdmin{},
		subnetService: &services.SubnetAdmin{},
	}
}

func (a *IpGroupAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing IpGroup filters: %+v", filtersMap)

	filters, err := ParseFilters[IpGroupFilters](c)
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
	if len(filters.UUIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
		logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *IpGroupAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting IpGroup list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	// 调用 service 层
	total, ipGroups, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d IpGroups (total: %d)", len(ipGroups), total)

	// 返回响应
	ipGroupListResp := &IpGroupListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(ipGroups),
	}

	// 构建响应
	ipGroupListResp.IpGroups = make([]*IpGroupResponse, ipGroupListResp.Limit)
	for i, ipGroup := range ipGroups {
		ipGroupListResp.IpGroups[i], err = a.getIpGroupResponse(ctx, ipGroup)
		if err != nil {
			logger.Errorf("Failed to create ip group response: %v", err)
			return nil, err
		}
	}

	logger.Debugf("List ip group successfully: %+v", ipGroupListResp)
	return ipGroupListResp, nil
}

func (a *IpGroupAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting IpGroup get query with ID: %s", id)

	ctx := c.Request.Context()
	ipGroup, err := a.service.GetIpGroupByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getIpGroupResponse(ctx, ipGroup)
	if err != nil {
		return
	}

	logger.Debugf("Get IpGroup successfully: %+v", resp)
	return
}

func (a *IpGroupAdapter) getIpGroupResponse(ctx context.Context, ipGroup *model.IpGroup) (ipGroupResp *IpGroupResponse, err error) {
	owner := orgAdmin.GetOrgName(ctx, ipGroup.Owner)

	var names []string
	subnets := make([]*BaseReference, len(ipGroup.Subnets))
	var total, allocated, reserved, idle int64
	for i, subnet := range ipGroup.Subnets {
		names = append(names, subnet.Name)
		subnets[i] = &BaseReference{
			ID:   subnet.UUID,
			Name: subnet.Name,
		}
		var itemTotal, itemAllocated, itemReserved, itemIdle int64
		itemTotal, itemAllocated, itemReserved, itemIdle, err = a.subnetService.AddressStatistics(ctx, subnet)
		if err != nil {
			logger.Errorf("address statistics error, subnet=%s, err=%v", subnet.UUID, err)
			return nil, fmt.Errorf("failed to get address statistics for subnet %s: %w", subnet.UUID, err)
		}
		total += itemTotal
		allocated += itemAllocated
		reserved += itemReserved
		idle += itemIdle
	}
	ipGroup.SubnetNames = strings.Join(names, ",")

	var dictInfo *BaseReference

	if ipGroup.DictionaryType != nil {
		dictInfo = &BaseReference{
			ID:   ipGroup.DictionaryType.UUID,
			Name: ipGroup.DictionaryType.Name,
		}
	}

	// Build associated floating ips list
	var floatingIpRefs []*FloatingIpWithInfo
	for _, fip := range ipGroup.FloatingIPs {
		var vlan int64
		if fip.Subnet != nil {
			vlan = fip.Subnet.Vlan
		}
		floatingIpRefs = append(floatingIpRefs, &FloatingIpWithInfo{
			BaseReference: &BaseReference{
				ID:   fip.UUID,
				Name: fip.Name,
			},
			Vlan:       vlan,
			IPAddress:  fip.IPAddress,
			FipAddress: fip.FipAddress,
			Type:       fip.Type,
		})
	}

	ipGroupResp = &IpGroupResponse{
		ResourceReference: &ResourceReference{
			ID:        ipGroup.UUID,
			Name:      ipGroup.Name,
			Owner:     owner,
			CreatedAt: ipGroup.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: ipGroup.UpdatedAt.Format(TimeStringForMat),
		},
		Type:           ipGroup.Type,
		Dictionary:     dictInfo,
		SubnetNames:    ipGroup.SubnetNames,
		Subnets:        subnets,
		FloatingIps:    floatingIpRefs,
		IPCount:        total,
		IdleCount:      idle,
		ReservedCount:  reserved,
		AllocatedCount: allocated,
	}
	return
}
