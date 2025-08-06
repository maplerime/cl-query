/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: Subnet resource adapter
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

type SiteSubnetInfo struct {
	*ResourceReference
	Network string         `json:"network"`
	Netmask string         `json:"netmask"`
	Gateway string         `json:"gateway"`
	Start   string         `json:"start"`
	End     string         `json:"end"`
	Group   *BaseReference `json:"group,omitempty"`
	Vlan    int64          `json:"vlan"`
}

type SubnetFilters struct {
	HasIdleIP bool     `json:"has_idle_ip,omitempty"` // 存在空闲IP
	Name      string   `json:"name,omitempty"`        // 子网名称 / ip地址like
	UUIDs     []string `json:"uuids,omitempty"`       // 子网UUID
	VpcID     string   `json:"vpc_id,omitempty"`      // VPC ID
	GroupID   string   `json:"group_id,omitempty"`    // 组ID
	Types     []string `json:"types,omitempty"`       // 子网类型
}

type SubnetResponse struct {
	*ResourceReference
	Network        string             `json:"network"`
	Netmask        string             `json:"netmask"`
	Gateway        string             `json:"gateway"`
	Start          string             `json:"start"`
	End            string             `json:"end"`
	NameServer     string             `json:"dns,omitempty"`
	VPC            *ResourceReference `json:"vpc,omitempty"`
	Group          *ResourceReference `json:"group,omitempty"`
	Type           string             `json:"type"`
	Vlan           int                `json:"vlan,omitempty"`
	IPCount        int64              `json:"ip_count"`        // total
	IdleCount      int64              `json:"idle_count"`      // idle
	ReservedCount  int64              `json:"reserved_count"`  // reserved
	AllocatedCount int64              `json:"allocated_count"` // allocated
}

type SubnetListResponse struct {
	Offset  int               `json:"offset"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Subnets []*SubnetResponse `json:"subnets"`
}

type SubnetAdapter struct {
	BaseAdapter
	service        *services.SubnetAdmin
	routerService  *services.RouterAdmin
	ipGroupService *services.IpGroupAdmin
}

func NewSubnetAdapter() *SubnetAdapter {
	logger.Debug("Creating new Subnet adapter")
	return &SubnetAdapter{
		service:        &services.SubnetAdmin{},
		routerService:  &services.RouterAdmin{},
		ipGroupService: &services.IpGroupAdmin{},
	}
}

func (a *SubnetAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Subnet filters: %+v", filtersMap)
	ctx := c.Request.Context()

	filters, parseErr := ParseFilters[SubnetFilters](c)
	if parseErr != nil {
		return "", parseErr
	}

	var conditions []string

	// 名字/IP
	if filters.Name != "" {
		queryStr := fmt.Sprintf("(subnets.name like '%%%s%%' OR addresses.address like '%%%s%%')", filters.Name, filters.Name)
		conditions = append(conditions, queryStr)
	}

	// UUIDS
	if len(filters.UUIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("subnets.uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
		logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
	}

	// IP组
	if filters.GroupID != "" {
		ipGroup, groupErr := a.ipGroupService.GetIpGroupByUUID(ctx, filters.GroupID)
		if groupErr != nil {
			return "", groupErr
		}
		conditions = append(conditions, fmt.Sprintf("subnets.group_id = %d", ipGroup.ID))
		logger.Debugf("Added group_id filter: %d", ipGroup.ID)
	}

	// VPC
	if filters.VpcID != "" {
		router, routerErr := a.routerService.GetRouterByUUID(ctx, filters.VpcID)
		if routerErr != nil {
			return "", routerErr
		}
		conditions = append(conditions, fmt.Sprintf("subnets.router_id = %d", router.ID))
		logger.Debugf("Added router_id filter: %d", router.ID)
	}

	if len(filters.Types) > 0 {
		types := make([]string, len(filters.Types))
		for i, t := range filters.Types {
			types[i] = fmt.Sprintf("'%s'", t)
		}
		conditions = append(conditions, fmt.Sprintf("subnets.type IN (%s)", strings.Join(types, ",")))
		logger.Debugf("Added types filter: %v", filters.Types)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return query, nil
}

func (a *SubnetAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Subnet list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	hasIdleIP := false
	if h, ok := req.Filters["has_idle_ip"]; ok {
		if hasIdleIP, ok = h.(bool); !ok {
			logger.Errorf("Invalid has_idle_ip filter value: %v", h)
			return nil, fmt.Errorf("invalid has_idle_ip filter value: %v", h)
		}
	}

	// 调用 service 层
	total, subnets, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query, hasIdleIP)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Subnets (total: %d)", len(subnets), total)

	subnetListResp := &SubnetListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(subnets),
	}
	subnetListResp.Subnets = make([]*SubnetResponse, subnetListResp.Limit)
	for i, subnet := range subnets {
		subnetListResp.Subnets[i], err = a.getSubnetResponse(ctx, subnet)
		if err != nil {
			return nil, err
		}
	}
	return subnetListResp, nil
}

func (a *SubnetAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting Subnet get query with ID: %s", id)

	ctx := c.Request.Context()
	subnet, err := a.service.GetSubnetByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getSubnetResponse(ctx, subnet)
	if err != nil {
		return
	}
	return
}

func (a *SubnetAdapter) getSubnetResponse(ctx context.Context, subnet *model.Subnet) (subnetResp *SubnetResponse, err error) {
	owner := orgAdmin.GetOrgName(ctx, subnet.Owner)
	subnetResp = &SubnetResponse{
		ResourceReference: &ResourceReference{
			ID:        subnet.UUID,
			Name:      subnet.Name,
			Owner:     owner,
			CreatedAt: subnet.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: subnet.UpdatedAt.Format(TimeStringForMat),
		},
		Network:    subnet.Network,
		Netmask:    subnet.Netmask,
		Gateway:    subnet.Gateway,
		NameServer: subnet.NameServer,
		Type:       subnet.Type,
	}
	if subnet.Router != nil {
		router := subnet.Router
		subnetResp.VPC = &ResourceReference{
			ID:   router.UUID,
			Name: router.Name,
		}
	}
	if subnet.Group != nil {
		group := subnet.Group
		subnetResp.Group = &ResourceReference{
			ID:   group.UUID,
			Name: group.Name,
		}
	}
	var total, allocated, reserved, idle int64
	total, allocated, reserved, idle, err = a.service.AddressStatistics(ctx, subnet)
	if err != nil {
		logger.Errorf("Failed to count addresses for subnet, err=%v", err)
		return
	}
	subnetResp.IPCount = total
	subnetResp.AllocatedCount = allocated
	subnetResp.ReservedCount = reserved
	subnetResp.IdleCount = idle

	return
}
