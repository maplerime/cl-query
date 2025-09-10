/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    claude - Initial implementation
 *
 *
 * Purpose: Instance IP Tree API
 *
**/

package api

import (
	"context"
	"github.com/maplerime/cl-query/utils/logging"
	"net/http"
	"strings"
	"web/src/model"

	"github.com/gin-gonic/gin"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

var logger = logging.MustGetLogger("api")

type IPTreePublicItem struct {
	SubnetID   string   `json:"subnet_id,omitempty"`
	SubnetName string   `json:"subnet_name,omitempty"`
	IPs        []string `json:"ips,omitempty"`
}

type IPTreeResponse struct {
	VPC     *ResourceReference  `json:"vpc,omitempty"`
	Private []string            `json:"private,omitempty"`
	Public  []*IPTreePublicItem `json:"public,omitempty"`
}

// IPTreeAPI IP树API结构体
type IPTreeAPI struct {
	instanceService   *services.InstanceAdmin
	floatingIpService *services.FloatingIpAdmin
	subnetService     *services.SubnetAdmin
}

// NewIPTreeAPI 创建IP树API实例
func NewIPTreeAPI() *IPTreeAPI {
	return &IPTreeAPI{
		instanceService:   &services.InstanceAdmin{},
		floatingIpService: &services.FloatingIpAdmin{},
		subnetService:     &services.SubnetAdmin{},
	}
}

// GetInstanceIPTree 获取实例的IP树
// @Summary 获取实例IP树结构
// @Description 获取指定实例的IP树结构，包含VPC、内网IP、浮动IP和站群IP等信息
// @Tags 实例
// @Produce json
// @Param id path string true "实例ID"
// @Success 200 {object} IPTreeResponse "IP树结构"
// @Failure 400 {object} APIError "请求参数错误"
// @Failure 404 {object} APIError "实例不存在"
// @Failure 500 {object} APIError "服务器内部错误"
// @Router /instances/{id}/subnet-ip-tree [get]
func (api *IPTreeAPI) GetInstanceIPTree(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		ErrorResponse(c, http.StatusBadRequest, "Instance ID is required", nil)
		return
	}

	// 解析ID，支持UUID或数字ID
	var instance *model.Instance
	var err error
	ctx := c.Request.Context()

	// 获取实例详情
	instance, err = api.instanceService.GetInstanceByUUID(ctx, idStr)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid instance ID", err)
		return
	}
	c.JSON(http.StatusOK, api.buildIPTree(ctx, instance))
}

func (api *IPTreeAPI) buildIPTree(ctx context.Context, instance *model.Instance) (response *IPTreeResponse) {
	response = &IPTreeResponse{}

	// 1. 处理VPC信息
	if instance.RouterID > 0 && instance.Router != nil {
		router := instance.Router
		response.VPC = &ResourceReference{
			ID:   router.UUID,
			Name: router.Name,
		}
	}

	// 用于避免重复添加的映射，同时保存子网信息
	subnetInfoMap := make(map[string]struct {
		Name string
		IPs  []string
	})

	for _, iface := range instance.Interfaces {

		// 内网IP
		if iface.Address.Subnet.Type == "internal" {
			response.Private = append(response.Private, strings.Split(iface.Address.Address, "/")[0])
		}

		// 处理Floating IP
		if iface.PrimaryIf {
			if len(instance.FloatingIps) > 0 {
				for _, fip := range instance.FloatingIps {
					if fip.Type != "site" && fip.Subnet != nil {
						subnetKey := fip.Subnet.UUID
						if _, exists := subnetInfoMap[subnetKey]; !exists {
							subnetInfoMap[subnetKey] = struct {
								Name string
								IPs  []string
							}{
								Name: fip.Subnet.Name,
								IPs:  []string{},
							}
						}
						info := subnetInfoMap[subnetKey]
						info.IPs = append(info.IPs, fip.IPAddress)
						subnetInfoMap[subnetKey] = info
					}
				}
			}

			// 处理站群IP
			if len(iface.SiteSubnets) > 0 {
				for _, siteSubnet := range iface.SiteSubnets {
					subnetKey := siteSubnet.UUID
					addresses, err := api.subnetService.GetAddressesBySubnet(ctx, siteSubnet.ID)
					if err != nil {
						logger.Errorf("Failed to get addresses for site subnet %s: %v", siteSubnet.UUID, err)
						continue
					}

					if _, exists := subnetInfoMap[subnetKey]; !exists {
						subnetInfoMap[subnetKey] = struct {
							Name string
							IPs  []string
						}{
							Name: siteSubnet.Name,
							IPs:  []string{},
						}
					}

					info := subnetInfoMap[subnetKey]
					for _, addr := range addresses {
						if addr.Address != siteSubnet.Gateway {
							info.IPs = append(info.IPs, strings.Split(addr.Address, "/")[0])
						}
					}
					subnetInfoMap[subnetKey] = info
				}
			}
		}
	}

	for subnetID, info := range subnetInfoMap {
		response.Public = append(response.Public, &IPTreePublicItem{
			SubnetID:   subnetID,
			SubnetName: info.Name,
			IPs:        info.IPs,
		})
	}

	return response
}
