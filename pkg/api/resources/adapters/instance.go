/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Instance resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"web/src/model"

	"github.com/gin-gonic/gin"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

type InstanceResponse struct {
	*ResourceReference
	Hostname    string                `json:"hostname"`
	Status      string                `json:"status"`
	LoginPort   int                   `json:"login_port"`
	Interfaces  []*InterfaceResponse  `json:"interfaces"`
	Volumes     []*VolumeInfoResponse `json:"volumes"`
	Cpu         int32                 `json:"cpu"`
	Memory      int32                 `json:"memory"`
	Disk        int32                 `json:"disk"`
	Image       *ResourceReference    `json:"image"`
	PasswdLogin bool                  `json:"passwd_login"`
	ZoneName    string                `json:"zone_name"`
	ZoneRemark  string                `json:"zone_remark"`
	VPC         *ResourceReference    `json:"vpc,omitempty"`
	Hypervisor  *BaseReference        `json:"hypervisor,omitempty"`
	Reason      string                `json:"reason"`
	Username    string                `json:"username,omitempty"`
}

type InstanceListResponse struct {
	Offset    int                 `json:"offset"`
	Total     int                 `json:"total"`
	Limit     int                 `json:"limit"`
	Instances []*InstanceResponse `json:"instances"`
}

type InstanceFilters struct {
	Hostname        string   `json:"hostname,omitempty"`
	Status          string   `json:"status,omitempty" binding:"omitempty"`
	UUID            string   `json:"uuid,omitempty" binding:"omitempty,uuid"`
	UUIDs           []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	VpcID           string   `json:"vpc_id,omitempty" binding:"omitempty,uuid"`
	VpcIDs          []string `json:"vpc_ids,omitempty" binding:"omitempty,dive,uuid"`
	SecurityGroupID string   `json:"security_group_id,omitempty" binding:"omitempty,uuid"`
	Hyper           *int32   `json:"hyper,omitempty" binding:"omitempty"`
	Keyword         string   `json:"keyword,omitempty" binding:"omitempty"`
}

type InstanceAdapter struct {
	BaseAdapter
	service         *services.InstanceAdmin
	routerService   *services.RouterAdmin
	secgroupService *services.SecgroupAdmin
	hyperService    *services.HyperAdmin
}

var interfaceAdapter = NewInterfaceAdapter()

func NewInstanceAdapter() *InstanceAdapter {
	logger.Debug("Creating new Instance adapter")
	return &InstanceAdapter{
		service:         &services.InstanceAdmin{},
		routerService:   &services.RouterAdmin{},
		secgroupService: &services.SecgroupAdmin{},
		hyperService:    &services.HyperAdmin{},
	}
}

func (a *InstanceAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Instance filters: %+v", filtersMap)

	filters, err := ParseFilters[InstanceFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// uuid查询
	if filters.UUID != "" {
		conditions = append(conditions, fmt.Sprintf("instances.uuid = '%s'", filters.UUID))
		logger.Debugf("Added UUID filter: %s", filters.UUID)
	}

	// hostname查询
	if filters.Hostname != "" {
		conditions = append(conditions, fmt.Sprintf("instances.hostname like '%%%s%%'", filters.Hostname))
		logger.Debugf("Added hostname filter: %s", filters.Hostname)
	}

	// 状态查询
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("instances.status = '%s'", filters.Status))
		logger.Debugf("Added status filter: %s", filters.Status)
	}

	// uuids查询
	if len(filters.UUIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf("instances.uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
		logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
	}

	// vpc查询
	if filters.VpcID != "" {
		router := &model.Router{}
		router, err = a.routerService.GetRouterByUUID(c.Request.Context(), filters.VpcID)
		if err != nil {
			return
		}
		conditions = append(conditions, fmt.Sprintf("instances.router_id = %d", router.ID))
		logger.Debugf("Added VPC ID filter: %s -> router_id = %d", filters.VpcID, router.ID)
	}

	// vpc_ids 查询
	if len(filters.VpcIDs) > 0 {
		routerIDs := make([]string, len(filters.VpcIDs))
		for i, vpcID := range filters.VpcIDs {
			router := &model.Router{}
			router, err = a.routerService.GetRouterByUUID(c.Request.Context(), vpcID)
			if err != nil {
				return
			}
			routerIDs[i] = fmt.Sprintf("%d", router.ID)
		}
		conditions = append(conditions, fmt.Sprintf("instances.router_id IN (%s)", strings.Join(routerIDs, ",")))
		logger.Debugf("Added VPC IDs filter: %v", filters.VpcIDs)
	}

	// 安全组查询
	if filters.SecurityGroupID != "" {
		conditions = append(conditions, fmt.Sprintf("secgroup_ifaces.security_group_id = '%s'", filters.SecurityGroupID))
		logger.Debugf("Added security group filter: %s", filters.SecurityGroupID)
	}

	// hyper查询
	if filters.Hyper != nil {
		hyper := &model.Hyper{}
		hyper, err = hyperAdmin.GetHyperByHostid(c.Request.Context(), *filters.Hyper)
		if err != nil {
			logger.Errorf("Failed to get hyper by host id %d: %v", filters.Hyper, err)
			return
		}
		conditions = append(conditions, fmt.Sprintf("instances.hyper = %d", hyper.Hostid))
		logger.Debugf("Added hyper filter: %s", filters.Hyper)
	}
	if filters.Keyword != "" {
		keywordCondition := fmt.Sprintf(`(
			instances.hostname LIKE '%%%s%%' OR
			organizations.name LIKE '%%%s%%' OR
			interfaces.mac_addr LIKE '%%%s%%' OR
			addresses.address LIKE '%%%s%%' OR
			addresses2.address LIKE '%%%s%%' OR
			floating_ips.fip_address LIKE '%%%s%%' OR
			floating_ips.int_address LIKE '%%%s%%' OR
			site_addresses.address LIKE '%%%s%%'
		)`, filters.Keyword, filters.Keyword, filters.Keyword, filters.Keyword, filters.Keyword, filters.Keyword, filters.Keyword, filters.Keyword)
		conditions = append(conditions, keywordCondition)
		logger.Debugf("Added keyword filter: %s", filters.Keyword)
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

func (a *InstanceAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Instance list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	// 调用 service 层
	logger.Debugf("Calling service layer with query: %s", query)
	total, instances, err := a.service.ListWithJoins(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Instances (total: %d)", len(instances), total)

	// 返回响应
	instanceListResp := &InstanceListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(instances),
	}
	instanceList := make([]*InstanceResponse, instanceListResp.Limit)
	for i, instance := range instances {
		instanceList[i], err = a.getInstanceResponse(ctx, instance)
		if err != nil {
			logger.Errorf("Failed to create instance response, %+v", err)
			ErrorResponse(c, http.StatusInternalServerError, "Internal error", err)
			return nil, err
		}
	}
	instanceListResp.Instances = instanceList
	logger.Debugf("List instances successfully, %+v", instanceListResp)
	return instanceListResp, nil
}

func (a *InstanceAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting Instance get query with ID: %s", id)

	ctx := c.Request.Context()
	instance, err := a.service.GetInstanceByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getInstanceResponse(ctx, instance)
	if err != nil {
		return
	}

	logger.Debugf("Get instance successfully: %+v", resp)
	return
}

func (a *InstanceAdapter) getInstanceResponse(ctx context.Context, instance *model.Instance) (instanceResp *InstanceResponse, err error) {
	logger.Debugf("Create instance response for instance %+v", instance)

	var owner string
	if instance.OwnerInfo != nil {
		owner = instance.OwnerInfo.Name
	} else {
		owner = orgAdmin.GetOrgName(ctx, instance.Owner)
	}

	instanceResp = &InstanceResponse{
		ResourceReference: &ResourceReference{
			ID:        instance.UUID,
			Owner:     owner,
			CreatedAt: instance.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: instance.UpdatedAt.Format(TimeStringForMat),
		},
		Hostname:    instance.Hostname,
		LoginPort:   int(instance.LoginPort),
		Status:      string(instance.Status),
		Reason:      instance.Reason,
		Cpu:         instance.Cpu,
		Memory:      instance.Memory,
		Disk:        instance.Disk,
		Username:    instance.Image.UserName,
		PasswdLogin: instance.PasswdLogin,
	}
	if instance.Image != nil {
		instanceResp.Image = &ResourceReference{
			ID:   instance.Image.UUID,
			Name: instance.Image.Name,
		}
	}
	if instance.Flavor != nil {
		instanceResp.Cpu = instance.Flavor.Cpu
		instanceResp.Memory = instance.Flavor.Memory
		instanceResp.Disk = instance.Flavor.Disk
	}
	if instance.Zone != nil {
		instanceResp.ZoneName = instance.Zone.Name
		instanceResp.ZoneRemark = instance.Zone.Remark
	}
	volumes := make([]*VolumeInfoResponse, len(instance.Volumes))
	for i, volume := range instance.Volumes {
		volumes[i] = &VolumeInfoResponse{
			ResourceReference: &ResourceReference{
				ID:   volume.UUID,
				Name: volume.Name,
			},
			Target:  volume.Target,
			Booting: volume.Booting,
		}
	}
	instanceResp.Volumes = volumes
	hyper, hyperErr := hyperAdmin.GetHyperByHostid(ctx, instance.Hyper)
	if hyperErr == nil {
		instanceResp.Hypervisor = &BaseReference{
			ID:   strconv.Itoa(int(instance.Hyper)),
			Name: hyper.Hostname,
		}
	}
	interfaces := make([]*InterfaceResponse, len(instance.Interfaces))

	for i, iface := range instance.Interfaces {
		interfaces[i], err = interfaceAdapter.getInterfaceResponse(ctx, instance, iface)
	}
	instanceResp.Interfaces = interfaces
	if instance.RouterID > 0 && instance.Router != nil {
		router := instance.Router
		instanceResp.VPC = &ResourceReference{
			ID:   router.UUID,
			Name: router.Name,
		}
	}
	logger.Debugf("Create instance response success, %+v", instanceResp)
	return
}
