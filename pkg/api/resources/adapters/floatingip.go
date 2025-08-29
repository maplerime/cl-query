/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: FloatingIP resource adapter
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

type InstanceInfo struct {
	*ResourceReference
	Hostname string `json:"hostname"`
	Status   string `json:"status,omitempty"`
}

type TargetInterface struct {
	*ResourceReference
	IpAddress    string        `json:"ip_address"`
	FromInstance *InstanceInfo `json:"from_instance"`
	MacAddr      string        `json:"mac_address,omitempty"`
}

type FloatingIpInfo struct {
	*ResourceReference
	IpAddress  string         `json:"ip_address"`
	FipAddress string         `json:"fip_address"`
	Group      *BaseReference `json:"group,omitempty"`
	Vlan       int64          `json:"vlan,omitempty"`
	Type       string         `json:"type,omitempty"`
}

// FloatingIpFilters 浮动IP查询过滤器
type FloatingIpFilters struct {
	Name       string   `json:"name,omitempty"`
	InstanceID string   `json:"instance_id,omitempty"`
	UUIDs      []string `json:"uuids,omitempty"`
	IsIdle     *bool    `json:"is_idle,omitempty"` // 是否查询空闲的/非空闲的
	Type       string   `json:"type,omitempty"`
}

type FloatingIpResponse struct {
	*ResourceReference
	PublicIp        string           `json:"public_ip"`
	TargetInterface *TargetInterface `json:"target_interface,omitempty"`
	VPC             *BaseReference   `json:"vpc,omitempty"`
	Inbound         int32            `json:"inbound"`
	Outbound        int32            `json:"outbound"`
	Group           *BaseReference   `json:"group,omitempty"`
	Subnet          *SiteSubnetInfo  `json:"subnet,omitempty"`
}

type FloatingIpListResponse struct {
	Offset      int                   `json:"offset"`
	Total       int                   `json:"total"`
	Limit       int                   `json:"limit"`
	FloatingIps []*FloatingIpResponse `json:"floating_ips"`
}

type FloatingIPAdapter struct {
	BaseAdapter
	service         *services.FloatingIpAdmin
	instanceService *services.InstanceAdmin
}

func NewFloatingIPAdapter() *FloatingIPAdapter {
	logger.Debug("Creating new FloatingIP adapter")
	return &FloatingIPAdapter{
		service:         &services.FloatingIpAdmin{},
		instanceService: &services.InstanceAdmin{},
	}
}

func (a *FloatingIPAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing FloatingIP filters: %+v", filtersMap)
	ctx := c.Request.Context()

	filters, err := ParseFilters[FloatingIpFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	if len(filters.UUIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
		logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
	}

	if filters.Name != "" {
		// fip_address | int_address | name
		conditions = append(conditions, fmt.Sprintf("(fip_address like '%%%s%%' OR int_address like '%%%s%%' OR name like '%%%s%%')", filters.Name, filters.Name, filters.Name))
		logger.Debugf("Added name filter: %s", filters.Name)
	}

	if filters.InstanceID != "" {
		instance := &model.Instance{}
		instance, err = a.instanceService.GetInstanceByUUID(ctx, filters.InstanceID)
		if err != nil {
			return
		}
		conditions = append(conditions, fmt.Sprintf("instance_id = %d", instance.ID))
		logger.Debugf("Added instance_id filter: %d", instance.ID)
	}

	if filters.IsIdle != nil {
		if *filters.IsIdle == true {
			// 查找空闲没有挂载到实例的
			conditions = append(conditions, "instance_id = 0")
		} else {
			// 查找已经挂载到实例的
			conditions = append(conditions, "instance_id != 0")
		}
	}

	if filters.Type != "" {
		conditions = append(conditions, fmt.Sprintf("type = '%s'", filters.Type))
		logger.Debugf("Added type filter: %s", filters.Type)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *FloatingIPAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting FloatingIP list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	// 调用 service 层
	logger.Debugf("Calling service layer with query: %s", query)
	total, floatingIps, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query, "")
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d FloatingIPs (total: %d)", len(floatingIps), total)

	floatingIpListResp := &FloatingIpListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(floatingIps),
	}
	floatingIpListResp.FloatingIps = make([]*FloatingIpResponse, floatingIpListResp.Limit)
	for i, floatingIp := range floatingIps {
		floatingIpListResp.FloatingIps[i], err = a.getFloatingIpResponse(ctx, floatingIp)
		if err != nil {
			return nil, err
		}
	}
	return floatingIpListResp, nil
}

func (a *FloatingIPAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting floating ip get query with ID: %s", id)

	ctx := c.Request.Context()
	floatingIp, err := a.service.GetFloatingIpByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getFloatingIpResponse(ctx, floatingIp)
	if err != nil {
		return
	}

	logger.Debugf("Get floating ip successfully: %+v", resp)
	return
}

func (a *FloatingIPAdapter) getFloatingIpResponse(ctx context.Context, floatingIp *model.FloatingIp) (floatingIpResp *FloatingIpResponse, err error) {
	owner := orgAdmin.GetOrgName(ctx, floatingIp.Owner)
	floatingIpResp = &FloatingIpResponse{
		ResourceReference: &ResourceReference{
			ID:        floatingIp.UUID,
			Name:      floatingIp.Name,
			Owner:     owner,
			CreatedAt: floatingIp.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: floatingIp.UpdatedAt.Format(TimeStringForMat),
		},
		PublicIp: floatingIp.FipAddress,
		Inbound:  floatingIp.Inbound,
		Outbound: floatingIp.Outbound,
	}
	if floatingIp.Router != nil {
		floatingIpResp.VPC = &BaseReference{
			ID:   floatingIp.Router.UUID,
			Name: floatingIp.Router.Name,
		}
	}
	if floatingIp.Group != nil {
		floatingIpResp.Group = &BaseReference{
			ID:   floatingIp.Group.UUID,
			Name: floatingIp.Group.Name,
		}
	}
	if floatingIp.Subnet != nil {
		floatingIpResp.Subnet = &SiteSubnetInfo{
			ResourceReference: &ResourceReference{
				ID:   floatingIp.Subnet.UUID,
				Name: floatingIp.Subnet.Name,
			},
			Network: floatingIp.Subnet.Network,
			Netmask: floatingIp.Subnet.Netmask,
			Gateway: floatingIp.Subnet.Gateway,
			Start:   floatingIp.Subnet.Start,
			End:     floatingIp.Subnet.End,
			Vlan:    floatingIp.Subnet.Vlan,
		}
		if floatingIp.Subnet.Group != nil {
			floatingIpResp.Subnet.Group = &BaseReference{
				ID:   floatingIp.Subnet.Group.UUID,
				Name: floatingIp.Subnet.Group.Name,
			}
		}
	}
	if floatingIp.Instance != nil && len(floatingIp.Instance.Interfaces) > 0 {
		instance := floatingIp.Instance
		interIp := strings.Split(floatingIp.IntAddress, "/")[0]
		owner = orgAdmin.GetOrgName(ctx, instance.Owner)
		floatingIpResp.TargetInterface = &TargetInterface{
			ResourceReference: &ResourceReference{
				ID: instance.Interfaces[0].UUID,
			},
			IpAddress: interIp,
			FromInstance: &InstanceInfo{
				ResourceReference: &ResourceReference{
					ID:    instance.UUID,
					Owner: owner,
				},
				Hostname: instance.Hostname,
			},
		}
	}
	return
}
