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
	"net/http"
	"strings"
	"web/src/model"

	"github.com/maplerime/cl-query/utils/logging"

	"github.com/gin-gonic/gin"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

var logger = logging.MustGetLogger("api")

// IPTreeSubnetItem is one subnet group in the IP tree, shared by both the
// private and public sections of IPTreeResponse.
type IPTreeSubnetItem struct {
	SubnetID      string   `json:"subnet_id,omitempty"`
	SubnetName    string   `json:"subnet_name,omitempty"`
	SubnetNetwork string   `json:"subnet_network,omitempty"`
	IPs           []string `json:"ips,omitempty"`
}

// IPTreeResponse is the response body of the instance subnet-ip-tree API.
type IPTreeResponse struct {
	VPC     *ResourceReference  `json:"vpc,omitempty"`
	Private []*IPTreeSubnetItem `json:"private,omitempty"`
	Public  []*IPTreeSubnetItem `json:"public,omitempty"`
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
// @Tags Instance
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

// buildIPTree assembles the IP tree of an instance: VPC info, private IPs
// grouped by internal subnet, and public IPs (floating IPs and site subnet
// IPs) grouped by their subnets. It always returns a non-nil response.
func (api *IPTreeAPI) buildIPTree(ctx context.Context, instance *model.Instance) (response *IPTreeResponse) {
	// Entry: building IP tree for the instance
	logger.Debugf("buildIPTree started for instance %s", instance.UUID)
	response = &IPTreeResponse{}

	// 1. VPC info comes from the instance router
	if instance.RouterID > 0 && instance.Router != nil {
		router := instance.Router
		response.VPC = &ResourceReference{
			ID:   router.UUID,
			Name: router.Name,
		}
	}

	// Both private and public IPs are grouped by subnet UUID
	privateMap := make(map[string]*IPTreeSubnetItem)
	publicMap := make(map[string]*IPTreeSubnetItem)

	// appendSubnetIP appends ip to the subnet entry in m, creating the entry
	// with subnet metadata on first use
	appendSubnetIP := func(m map[string]*IPTreeSubnetItem, subnet *model.Subnet, ip string) {
		item, exists := m[subnet.UUID]
		if !exists {
			item = &IPTreeSubnetItem{
				SubnetID:      subnet.UUID,
				SubnetName:    subnet.Name,
				SubnetNetwork: subnet.Network,
			}
			m[subnet.UUID] = item
		}
		item.IPs = append(item.IPs, ip)
	}

	// Walk all interfaces to collect private and public IPs
	for _, iface := range instance.Interfaces {

		// Private IPs: grouped by their internal subnet
		if iface.Address.Subnet.Type == "internal" {
			appendSubnetIP(privateMap, iface.Address.Subnet, strings.Split(iface.Address.Address, "/")[0])
		}

		// Floating IPs and site subnets only attach to the primary interface
		if iface.PrimaryIf {
			if len(instance.FloatingIps) > 0 {
				for _, fip := range instance.FloatingIps {
					// Site-type floating IPs are covered by the site subnet branch below
					if fip.Type != "site" && fip.Subnet != nil {
						appendSubnetIP(publicMap, fip.Subnet, fip.IPAddress)
					}
				}
			}

			// Site subnet IPs: all addresses of the subnet except the gateway
			if len(iface.SiteSubnets) > 0 {
				for _, siteSubnet := range iface.SiteSubnets {
					addresses, err := api.subnetService.GetAddressesBySubnet(ctx, siteSubnet.ID)
					if err != nil {
						// Skip this site subnet but keep building the rest of the tree
						logger.Errorf("Failed to get addresses for site subnet %s: %v", siteSubnet.UUID, err)
						continue
					}

					for _, addr := range addresses {
						// The gateway address is not usable by the instance
						if addr.Address != siteSubnet.Gateway {
							appendSubnetIP(publicMap, siteSubnet, strings.Split(addr.Address, "/")[0])
						}
					}
				}
			}
		}
	}

	// Flatten the grouping maps into the response arrays
	for _, item := range privateMap {
		response.Private = append(response.Private, item)
	}
	for _, item := range publicMap {
		response.Public = append(response.Public, item)
	}

	// Exit: report how many subnet groups were built
	logger.Debugf("buildIPTree finished for instance %s: %d private subnets, %d public subnets",
		instance.UUID, len(response.Private), len(response.Public))
	return response
}
