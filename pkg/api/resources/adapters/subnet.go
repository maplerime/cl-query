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
	IPCount int64          `json:"ip_count"`
}

type SubnetFilters struct {
	HasIdleIP bool     `json:"has_idle_ip,omitempty"` // 存在空闲IP
	Name      string   `json:"name,omitempty"`        // 子网名称 / ip地址like
	UUIDs     []string `json:"uuids,omitempty"`       // 子网UUID
	VpcID     string   `json:"vpc_id,omitempty"`      // VPC ID
	GroupID   string   `json:"group_id,omitempty"`    // 组ID
	Types     []string `json:"types,omitempty"`       // 子网类型
	NoGroup   bool     `json:"no_group,omitempty"`    // 是否不属于任何组
	Vlan      *int64   `json:"vlan,omitempty"`        // VLAN ID
	SubType1  string   `json:"sub_type1,omitempty"`   // 地区（dictionaries.sub_type1）
	SubType2  string   `json:"sub_type2,omitempty"`   // 线路（dictionaries.sub_type2）
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
	Priority       int32              `json:"priority"`
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

var subnetOrderMap = map[string]string{
	"priority":         "subnets.priority ASC, subnets.id ASC",
	"-priority":        "subnets.priority DESC, subnets.id ASC",
	"ip_count":         "stat_ip_count ASC, subnets.id ASC",
	"-ip_count":        "stat_ip_count DESC, subnets.id ASC",
	"allocated_count":  "stat_allocated_count ASC, subnets.id ASC",
	"-allocated_count": "stat_allocated_count DESC, subnets.id ASC",
	"reserved_count":   "stat_reserved_count ASC, subnets.id ASC",
	"-reserved_count":  "stat_reserved_count DESC, subnets.id ASC",
	"idle_count":       "stat_idle_count ASC, subnets.id ASC",
	"-idle_count":      "stat_idle_count DESC, subnets.id ASC",
}

func subnetOrderClause(order string) string {
	if clause, ok := subnetOrderMap[strings.TrimSpace(order)]; ok {
		return clause
	}
	return order
}

func subnetNameCondition(name string) string {
	return fmt.Sprintf(
		"(subnets.name LIKE '%%%s%%' OR EXISTS (SELECT 1 FROM addresses a2 "+
			"WHERE a2.subnet_id = subnets.id AND a2.deleted_at IS NULL AND a2.address LIKE '%%%s%%'))",
		name, name)
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
		conditions = append(conditions, subnetNameCondition(filters.Name))
	}

	// UUIDS
	if filters.UUIDs != nil {
		if len(filters.UUIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("subnets.uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
			logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
		} else {
			conditions = append(conditions, "1=0")
		}
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

	// NoGroup
	if filters.NoGroup {
		conditions = append(conditions, "subnets.group_id = 0")
		logger.Debug("Added no_group filter")
	}

	// Vlan
	if filters.Vlan != nil {
		conditions = append(conditions, fmt.Sprintf("subnets.vlan = %d", *filters.Vlan))
		logger.Debugf("Added vlan filter: %d", *filters.Vlan)
	}

	// 地区(sub_type1)+线路(sub_type2)：经 dictionaries -> ip_groups -> subnets 反查过滤。
	// 各自独立生效；都传则 AND；匹配不到字典时子查询为空集，自然返回空列表（无 fallback）。
	if filters.SubType1 != "" || filters.SubType2 != "" {
		var dictConds []string
		if filters.SubType1 != "" {
			dictConds = append(dictConds, fmt.Sprintf("d.sub_type1 = '%s'", filters.SubType1))
		}
		if filters.SubType2 != "" {
			dictConds = append(dictConds, fmt.Sprintf("d.sub_type2 = '%s'", filters.SubType2))
		}
		subQuery := fmt.Sprintf(
			"subnets.group_id IN (SELECT ig.id FROM ip_groups ig "+
				"JOIN dictionaries d ON ig.type_id = d.id "+
				"WHERE ig.type = 'system' AND d.deleted_at IS NULL AND ig.deleted_at IS NULL AND %s)",
			strings.Join(dictConds, " AND "))
		conditions = append(conditions, subQuery)
		logger.Debugf("Added sub_type filter: %s", subQuery)
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
		return nil, err
	}

	hasIdleIP := false
	if h, ok := req.Filters["has_idle_ip"]; ok {
		if hasIdleIP, ok = h.(bool); !ok {
			logger.Errorf("Invalid has_idle_ip filter value: %v", h)
			return nil, NewCLError(ErrInvalidParameter, "has_idle_ip filter must be a boolean", nil)
		}
	}

	// 调用 service 层
	req.Order = subnetOrderClause(req.Order)
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
	for i, item := range subnets {
		subnetListResp.Subnets[i], err = a.getSubnetResponse(ctx, item.Subnet, item.Stats)
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

	resp, err = a.getSubnetResponse(ctx, subnet, nil)
	if err != nil {
		return
	}
	return
}

func (a *SubnetAdapter) getSubnetResponse(ctx context.Context, subnet *model.Subnet, stats *services.SubnetStats) (subnetResp *SubnetResponse, err error) {
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
		Start:      subnet.Start,
		End:        subnet.End,
		NameServer: subnet.NameServer,
		Type:       subnet.Type,
		Vlan:       int(subnet.Vlan),
		Priority:   subnet.Priority,
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
	if stats == nil {
		var total, allocated, reserved, idle int64
		total, allocated, reserved, idle, err = a.service.AddressStatistics(ctx, subnet)
		if err != nil {
			logger.Errorf("Failed to count addresses for subnet, err=%v", err)
			return
		}
		stats = &services.SubnetStats{
			IPCount:        total,
			AllocatedCount: allocated,
			ReservedCount:  reserved,
			IdleCount:      idle,
		}
	}
	subnetResp.IPCount = stats.IPCount
	subnetResp.AllocatedCount = stats.AllocatedCount
	subnetResp.ReservedCount = stats.ReservedCount
	subnetResp.IdleCount = stats.IdleCount

	return
}
