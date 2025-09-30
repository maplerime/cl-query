/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: user resource usage api
 *
**/

package api

import (
	"github.com/gin-gonic/gin"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
	"net/http"
)

type UsageAPI struct{}

func NewUsageAPI() *UsageAPI {
	return &UsageAPI{}
}

type UsageSummaryResponse struct {
	InstanceCount      int64 `json:"instance_count"`
	CpuTotal           int64 `json:"cpu_total"`
	MemTotal           int64 `json:"mem_total"`
	InterfaceCount     int64 `json:"interface_count"`
	DiskCount          int64 `json:"disk_count"`
	DiskSize           int64 `json:"disk_size"` // in MB
	VpcCount           int64 `json:"vpc_count"`
	SubnetCount        int64 `json:"subnet_count"`
	FloatingIpCount    int64 `json:"floating_ip_count"`
	SecurityGroupCount int64 `json:"security_group_count"`
	SecurityRuleCount  int64 `json:"security_rule_count"`
}

// Summary
// @Summary 获取用量统计
// @Description 获取用量统计
// @Tags User
// @Produce json
// @Param id path string true "实例ID"
// @Success 200 {object} IPTreeResponse "IP树结构"
// @Failure 400 {object} APIError "请求参数错误"
// @Failure 404 {object} APIError "实例不存在"
// @Failure 500 {object} APIError "服务器内部错误"
// @Router /usage/summary [get]
func (api *UsageAPI) Summary(c *gin.Context) {
	ctx := c.Request.Context()

	instanceAdmin := &services.InstanceAdmin{}
	interfaceAdmin := &services.InterfaceAdmin{}
	volumeAdmin := &services.VolumeAdmin{}
	routerAdmin := &services.RouterAdmin{}
	subnetAdmin := &services.SubnetAdmin{}
	floatingIpAdmin := &services.FloatingIpAdmin{}
	securityGroupAdmin := &services.SecgroupAdmin{}
	securityRuleAdmin := &services.SecruleAdmin{}

	instanceCount, err := instanceAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get instance count", err)
		return
	}
	cpuTotal, err := instanceAdmin.SumCPU(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get CPU total", err)
		return
	}
	memTotal, err := instanceAdmin.SumMemory(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get memory total", err)
		return
	}
	interfaceCount, err := interfaceAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get interface count", err)
		return
	}
	diskCount, err := volumeAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get disk count", err)
		return
	}
	diskSize, err := volumeAdmin.SumSize(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get disk size", err)
		return
	}
	vpcCount, err := routerAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get VPC count", err)
		return
	}
	subnetCount, err := subnetAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get subnet count", err)
		return
	}
	floatingIpCount, err := floatingIpAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get floating IP count", err)
		return
	}
	securityGroupCount, err := securityGroupAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get security group count", err)
		return
	}
	securityRuleCount, err := securityRuleAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get security rule count", err)
		return
	}

	c.JSON(http.StatusOK, UsageSummaryResponse{
		InstanceCount:      instanceCount,
		CpuTotal:           cpuTotal,
		MemTotal:           memTotal,
		InterfaceCount:     interfaceCount,
		DiskCount:          diskCount,
		DiskSize:           diskSize * 1024, // convert to MB
		VpcCount:           vpcCount,
		SubnetCount:        subnetCount,
		FloatingIpCount:    floatingIpCount,
		SecurityGroupCount: securityGroupCount,
		SecurityRuleCount:  securityRuleCount,
	})
}
