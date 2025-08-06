/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: SecurityGroupRule resource adapter
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

type SecruleResponse struct {
	*ResourceReference
	RemoteCIDR  string         `json:"remote_cidr,omitempty"`
	RemoteGroup *BaseReference `json:"remote_group,omitempty"`
	Direction   string         `json:"direction"`
	IpVersion   string         `json:"ip_version"`
	Protocol    string         `json:"protocol"`
	PortMin     int32          `json:"port_min"`
	PortMax     int32          `json:"port_max"`
}

type SecruleListResponse struct {
	Offset        int                `json:"offset"`
	Total         int                `json:"total"`
	Limit         int                `json:"limit"`
	SecurityRules []*SecruleResponse `json:"security_rules"`
}

type SecruleFilters struct {
	SecGroupID string `json:"security_group_id" binding:"required,uuid"`
	Direction  string `json:"direction,omitempty" binding:"omitempty,oneof=ingress egress"`
}

type SecruleAdapter struct {
	BaseAdapter
	service         *services.SecruleAdmin
	secgroupService *services.SecgroupAdmin
}

func NewSecruleAdapter() *SecruleAdapter {
	logger.Debug("Creating new Secrule adapter")
	return &SecruleAdapter{
		service:         &services.SecruleAdmin{},
		secgroupService: &services.SecgroupAdmin{},
	}
}

func (a *SecruleAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing SecurityRule filters: %+v", filtersMap)

	filters, err := ParseFilters[SecruleFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// 安全组查询
	secgroup, err := a.secgroupService.GetSecgroupByUUID(c.Request.Context(), filters.SecGroupID)
	if err != nil {
		return
	}
	conditions = append(conditions, fmt.Sprintf("secgroup = %d", secgroup.ID))
	logger.Debugf("Added security group filter: secgroup = %d", secgroup.ID)

	// direction 查询
	if filters.Direction != "" {
		conditions = append(conditions, fmt.Sprintf("direction = '%s'", filters.Direction))
		logger.Debugf("Added direction filter: %s", filters.Direction)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *SecruleAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Secrule list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	// 调用 service 层
	total, secrules, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Secrules (total: %d)", len(secrules), total)

	// 返回响应
	secruleListResp := &SecruleListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(secrules),
	}
	secruleListResp.SecurityRules = make([]*SecruleResponse, secruleListResp.Limit)
	for i, secrule := range secrules {
		secruleListResp.SecurityRules[i], err = a.getSecruleResponse(ctx, secrule)
		if err != nil {
			return nil, err
		}
	}
	logger.Debugf("List secrules successfully: %+v", secruleListResp)
	return secruleListResp, nil
}

func (a *SecruleAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting SecurityRule get query with ID: %s", id)

	ctx := c.Request.Context()
	rule, err := a.service.GetSecruleByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getSecruleResponse(ctx, rule)
	if err != nil {
		return
	}

	logger.Debugf("Get rule successfully: %+v", resp)
	return
}

func (a *SecruleAdapter) getSecruleResponse(ctx context.Context, secrule *model.SecurityRule) (*SecruleResponse, error) {
	secruleResp := &SecruleResponse{
		ResourceReference: &ResourceReference{
			ID: secrule.UUID,
			//Name: secrule.Name,
			CreatedAt: secrule.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: secrule.UpdatedAt.Format(TimeStringForMat),
		},
		PortMin:   secrule.PortMin,
		PortMax:   secrule.PortMax,
		Direction: secrule.Direction,
		IpVersion: secrule.IpVersion,
		Protocol:  secrule.Protocol,
	}

	if secrule.RemoteIp != "" {
		secruleResp.RemoteCIDR = secrule.RemoteIp
	} else if secrule.RemoteGroup != nil {
		secruleResp.RemoteGroup = &BaseReference{
			ID:   secrule.RemoteGroup.UUID,
			Name: secrule.RemoteGroup.Name,
		}
	}

	return secruleResp, nil
}
