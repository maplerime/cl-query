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
	Statuses        []string `json:"statuses,omitempty" binding:"omitempty"`
	UUID            string   `json:"uuid,omitempty" binding:"omitempty,uuid"`
	UUIDs           []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	VpcID           string   `json:"vpc_id,omitempty" binding:"omitempty,uuid"`
	VpcIDs          []string `json:"vpc_ids,omitempty" binding:"omitempty,dive,uuid"`
	SecurityGroupID string   `json:"security_group_id,omitempty" binding:"omitempty,uuid"`
	Hyper           *int32   `json:"hyper,omitempty" binding:"omitempty"`
	Hypers          []int32  `json:"hypers,omitempty" binding:"omitempty"`
	Keyword         string   `json:"keyword,omitempty" binding:"omitempty"`
	AdjustRuleID    string   `json:"adjust_rule_id,omitempty" binding:"omitempty,uuid"`
	AssignableIpID  string   `json:"assignable_ip_id,omitempty" binding:"omitempty,uuid"`
}

type InstanceAdapter struct {
	BaseAdapter
	service           *services.InstanceAdmin
	routerService     *services.RouterAdmin
	secgroupService   *services.SecgroupAdmin
	hyperService      *services.HyperAdmin
	ipGroupService    *services.IpGroupAdmin
	floatingIpService *services.FloatingIpAdmin
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
	ctx := c.Request.Context()

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

	// 状态批量查询
	if filters.Statuses != nil && len(filters.Statuses) > 0 {
		conditions = append(conditions, fmt.Sprintf("instances.status IN ('%s')", strings.Join(filters.Statuses, "','")))
		logger.Debugf("Added status batch filter: %v", filters.Statuses)
	}

	// uuids查询
	if filters.UUIDs != nil {
		if len(filters.UUIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("instances.uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
			logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
		} else {
			conditions = append(conditions, "1=0")
		}
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
		// 转化为安全组的id
		secgroup := &model.SecurityGroup{}
		secgroup, err = a.secgroupService.GetSecgroupByUUID(ctx, filters.SecurityGroupID)
		if err != nil {
			return
		}
		conditions = append(conditions, fmt.Sprintf("secgroup_ifaces.security_group_id = %d", secgroup.ID))
		logger.Debugf("Added security group filter: %s", filters.SecurityGroupID)
	}

	// 智能规则查询
	if filters.AdjustRuleID != "" {
		conditions = append(conditions, fmt.Sprintf("adjust_rule_group.rule_id = '%s'", filters.AdjustRuleID))
		logger.Debugf("Added adjust rule ID filter: %s", filters.AdjustRuleID)
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

	// hypers查询
	if len(filters.Hypers) > 0 {
		var hyperIDs []string
		for _, hostID := range filters.Hypers {
			_, err = hyperAdmin.GetHyperByHostid(c.Request.Context(), hostID)
			if err != nil {
				logger.Errorf("Failed to get hyper by host id %d: %v", hostID, err)
				return
			}
			hyperIDs = append(hyperIDs, strconv.FormatInt(int64(hostID), 10))
		}

		conditions = append(
			conditions,
			fmt.Sprintf("instances.hyper IN (%s)", strings.Join(hyperIDs, ",")),
		)

		logger.Debugf("Added hyper filter: %v", hyperIDs)
	}

	// 可以分配到这个IP上的实例
	if filters.AssignableIpID != "" {
		ip := &model.FloatingIp{}
		ip, err = a.floatingIpService.GetFloatingIpByUUID(ctx, filters.AssignableIpID)
		if err != nil {
			return
		}
		if ip.Subnet != nil && ip.Subnet.Group != nil {
			subnetGroup := ip.Subnet.Group
			condition := fmt.Sprintf(`(
				(floating_ips.id IS NOT NULL AND fip_subnet.group_id = %d) OR
				floating_ips.id IS NULL
			)`, subnetGroup.ID)
			conditions = append(conditions, condition)
			logger.Debugf("Added assignable IP filter: instances with floating IP in group %d or no floating IP", subnetGroup.ID)
		}
	}

	// 关键字查询
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
		return nil, err
	}

	// 调用 service 层
	logger.Debugf("Calling service layer with query: %s", query)
	total, instances, err := a.service.ListWithJoins(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Instances (total: %d)", len(instances), total)

	// 批量查询所有需要的hyper信息
	hyperIds := make([]int32, 0, len(instances))
	for _, instance := range instances {
		hyperIds = append(hyperIds, instance.Hyper)
	}
	hypersMap, err := a.hyperService.GetHypersByHostids(ctx, hyperIds)
	if err != nil {
		logger.Errorf("Failed to batch query hypers: %v", err)
		// 如果批量查询失败，设置为nil，会fallback到单次查询
		hypersMap = nil
	}

	// 返回响应
	instanceListResp := &InstanceListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(instances),
	}
	instanceList := make([]*InstanceResponse, instanceListResp.Limit)
	for i, instance := range instances {
		instanceList[i], err = a.getInstanceResponse(ctx, instance, hypersMap)
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

func (a *InstanceAdapter) getInstanceResponse(ctx context.Context, instance *model.Instance, hypersMap ...map[int32]*model.Hyper) (instanceResp *InstanceResponse, err error) {
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
	volumes := make([]*VolumeInfoResponse, len(instance.Volumes))
	for i, volume := range instance.Volumes {
		volumes[i] = &VolumeInfoResponse{
			ResourceReference: &ResourceReference{
				ID:   volume.UUID,
				Name: volume.Name,
			},
			Target:  volume.Target,
			Booting: volume.Booting,
			Path:    volume.Path,
			PoolID:  volume.PoolID,
		}
	}
	instanceResp.Volumes = volumes

	// 优化hyper查询：使用批量查询结果或单次查询
	var hyper *model.Hyper
	var hyperErr error

	if len(hypersMap) > 0 && hypersMap[0] != nil {
		// 使用批量查询的结果
		if h, exists := hypersMap[0][instance.Hyper]; exists {
			hyper = h
		} else {
			hyperErr = fmt.Errorf("hyper not found in batch result")
		}
	} else {
		// 使用原有的单次查询方式
		hyper, hyperErr = hyperAdmin.GetHyperByHostid(ctx, instance.Hyper)
	}

	if hyperErr == nil {
		instanceResp.Hypervisor = &BaseReference{
			ID:   strconv.Itoa(int(instance.Hyper)),
			Name: hyper.Hostname,
		}
		if hyper.Zone != nil {
			instanceResp.ZoneName = hyper.Zone.Name
			instanceResp.ZoneRemark = hyper.Zone.Remark
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
