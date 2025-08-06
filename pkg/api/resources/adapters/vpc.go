/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: VPC resource adapter
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

var subnetAdapter = NewSubnetAdapter()

type VPCResponse struct {
	*ResourceReference
	Subnets []*SubnetResponse `json:"subnets,omitempty"`
}

type VPCListResponse struct {
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	VPCs   []*VPCResponse `json:"vpcs"`
}

type VPCFilters struct {
	UUIDs []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	Name  string   `json:"name,omitempty"`
}

type VPCAdapter struct {
	BaseAdapter
	service       *services.RouterAdmin
	subnetService *services.SubnetAdmin
}

func NewVPCAdapter() *VPCAdapter {
	logger.Debug("Creating new VPC adapter")
	return &VPCAdapter{
		service:       &services.RouterAdmin{},
		subnetService: &services.SubnetAdmin{},
	}
}

func (a *VPCAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing VPC filters: %+v", filtersMap)

	filters, err := ParseFilters[VPCFilters](c)
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

func (a *VPCAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting VPC list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	// 调用 service 层
	total, routers, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d VPCs (total: %d)", len(routers), total)

	// 返回响应
	vpcListResp := &VPCListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(routers),
	}
	vpcListResp.VPCs = make([]*VPCResponse, vpcListResp.Limit)
	for i, router := range routers {
		vpcListResp.VPCs[i], err = a.getVPCResponse(ctx, router)
		if err != nil {
			return nil, err
		}
	}

	logger.Debugf("List VPCs successfully: %+v", vpcListResp)
	return vpcListResp, nil
}

func (a *VPCAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting VPC get query with ID: %s", id)

	ctx := c.Request.Context()
	router, err := a.service.GetRouterByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getVPCResponse(ctx, router)
	if err != nil {
		return
	}

	return
}

func (a *VPCAdapter) getVPCResponse(ctx context.Context, router *model.Router) (vpcResp *VPCResponse, err error) {
	owner := orgAdmin.GetOrgName(ctx, router.Owner)
	vpcResp = &VPCResponse{
		ResourceReference: &ResourceReference{
			ID:        router.UUID,
			Name:      router.Name,
			Owner:     owner,
			CreatedAt: router.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: router.UpdatedAt.Format(TimeStringForMat),
		},
	}
	vpcResp.Subnets = make([]*SubnetResponse, len(router.Subnets))
	for i, subnet := range router.Subnets {
		vpcResp.Subnets[i], err = subnetAdapter.getSubnetResponse(ctx, subnet)
		if err != nil {
			return
		}
	}
	return
}
