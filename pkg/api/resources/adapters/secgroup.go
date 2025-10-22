/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: SecurityGroup resource adapter
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

type SecurityGroupResponse struct {
	*ResourceReference
	IsDefault        bool               `json:"is_default"`
	VPC              *ResourceReference `json:"vpc,omitempty"`
	TargetInterfaces []*TargetInterface `json:"target_interfaces,omitempty"`
	RuleCount        int64              `json:"rule_count"`
}

type SecurityGroupListResponse struct {
	Offset         int                      `json:"offset"`
	Total          int                      `json:"total"`
	Limit          int                      `json:"limit"`
	SecurityGroups []*SecurityGroupResponse `json:"security_groups"`
}

type SecurityGroupFilters struct {
	Name      string   `json:"name,omitempty"`
	UUIDs     []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	VpcID     string   `json:"vpc_id,omitempty" binding:"omitempty,uuid"`
	ExactName string   `json:"exact_name,omitempty"`
}

type SecurityGroupAdapter struct {
	BaseAdapter
	service         *services.SecgroupAdmin
	routerService   *services.RouterAdmin
	instanceService *services.InstanceAdmin
	ruleService     *services.SecruleAdmin
}

func NewSecurityGroupAdapter() *SecurityGroupAdapter {
	logger.Debug("Creating new SecurityGroup adapter")
	return &SecurityGroupAdapter{
		service:         &services.SecgroupAdmin{},
		routerService:   &services.RouterAdmin{},
		instanceService: &services.InstanceAdmin{},
		ruleService:     &services.SecruleAdmin{},
	}
}

func (a *SecurityGroupAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing SecurityGroup filters: %+v", filtersMap)

	filters, err := ParseFilters[SecurityGroupFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name like '%%%s%%'", filters.Name))
		logger.Debugf("Added name filter: %s", filters.Name)
	}

	// name精准匹配
	if filters.ExactName != "" {
		conditions = append(conditions, fmt.Sprintf("name = '%s'", filters.ExactName))
		logger.Debugf("Added exact name filter: %s", filters.ExactName)
	}

	if filters.UUIDs != nil {
		if len(filters.UUIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
			logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
		} else {
			conditions = append(conditions, "1=0")
		}
	}

	// 处理 vpc_id 过滤器 - 通过 vpc_id 获取递增 ID
	if filters.VpcID != "" {
		router := &model.Router{}
		router, err = a.routerService.GetRouterByUUID(c.Request.Context(), filters.VpcID)
		if err != nil {
			return
		}
		conditions = append(conditions, fmt.Sprintf("router_id = %d", router.ID))
		logger.Debugf("Added vpc_id filter: %s -> router_id = %d", filters.VpcID, router.ID)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *SecurityGroupAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting SecurityGroup list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	// 调用 service 层
	logger.Debugf("Calling service layer with query: %s", query)
	total, secgroups, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d SecurityGroups (total: %d)", len(secgroups), total)

	// 返回响应
	secgroupListResp := &SecurityGroupListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(secgroups),
	}
	secgroupList := make([]*SecurityGroupResponse, secgroupListResp.Limit)
	for i, secgroup := range secgroups {
		secgroupList[i], err = a.getSecurityGroupResponse(ctx, secgroup)
		if err != nil {
			logger.Errorf("Failed to create secgroup response: %v", err)
			return nil, err
		}
	}
	secgroupListResp.SecurityGroups = secgroupList
	logger.Debugf("List secgroups successfully: %+v", secgroupListResp)
	return secgroupListResp, nil
}

func (a *SecurityGroupAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting SecurityGroup get query with ID: %s", id)

	ctx := c.Request.Context()
	secgroup, err := a.service.GetSecgroupByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getSecurityGroupResponse(ctx, secgroup)
	if err != nil {
		return
	}

	logger.Debugf("Get secgroup successfully: %+v", resp)
	return
}

func (a *SecurityGroupAdapter) getSecurityGroupResponse(ctx context.Context, secgroup *model.SecurityGroup) (secgroupResp *SecurityGroupResponse, err error) {
	logger.Debugf("Create secgroup response for secgroup %+v", secgroup)

	owner := orgAdmin.GetOrgName(ctx, secgroup.Owner)

	// rule count
	ruleCount, err := a.ruleService.CountBySecGroup(ctx, secgroup.ID)
	if err != nil {
		logger.Errorf("Failed to count security group rules: %v", err)
		ruleCount = 0
		err = nil
	}

	secgroupResp = &SecurityGroupResponse{
		ResourceReference: &ResourceReference{
			ID:        secgroup.UUID,
			Name:      secgroup.Name,
			Owner:     owner,
			CreatedAt: secgroup.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: secgroup.UpdatedAt.Format(TimeStringForMat),
		},
		IsDefault: secgroup.IsDefault,
		RuleCount: ruleCount,
	}
	if secgroup.Router != nil {
		secgroupResp.VPC = &ResourceReference{
			ID:   secgroup.Router.UUID,
			Name: secgroup.Router.Name,
		}
	}
	err = a.service.GetSecgroupInterfaces(ctx, secgroup)
	if err != nil {
		return
	}
	for _, iface := range secgroup.Interfaces {
		targetIface := &TargetInterface{
			ResourceReference: &ResourceReference{
				ID: iface.UUID,
			},
		}
		if iface.Address != nil {
			targetIface.IpAddress = strings.Split(iface.Address.Address, "/")[0]
		}
		if iface.Instance > 0 {
			var instance *model.Instance
			instance, err = a.instanceService.Get(ctx, iface.Instance)
			if err != nil {
				err = nil
				continue
			}
			owner := orgAdmin.GetOrgName(ctx, instance.Owner)
			targetIface.FromInstance = &InstanceInfo{
				ResourceReference: &ResourceReference{
					ID:        instance.UUID,
					Owner:     owner,
					CreatedAt: instance.CreatedAt.Format(TimeStringForMat),
				},
				Hostname: instance.Hostname,
				Status:   string(instance.Status),
			}
		}
		secgroupResp.TargetInterfaces = append(secgroupResp.TargetInterfaces, targetIface)
	}

	logger.Debugf("Create secgroup response success: %+v", secgroupResp)
	return
}
