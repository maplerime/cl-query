/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: Interface resource adapter
 *
**/

package adapters

import (
	"context"
	"fmt"
	"net/http"
	"web/src/model"

	"github.com/gin-gonic/gin"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

type AddressInfo struct {
	IPAddress string          `json:"ip_address"`
	Subnet    *SubnetResponse `json:"subnet"`
}

type InterfaceResponse struct {
	*BaseReference
	*AddressInfo
	MacAddress         string               `json:"mac_address"`
	SecondaryAddresses []*AddressInfo       `json:"secondary_addresses,omitempty"`
	IsPrimary          bool                 `json:"is_primary"`
	Inbound            int32                `json:"inbound"`
	Outbound           int32                `json:"outbound"`
	SiteSubnets        []*SiteSubnetInfo    `json:"site_subnets,omitempty"`
	FloatingIps        []*FloatingIpInfo    `json:"floating_ips,omitempty"`
	SecurityGroups     []*ResourceReference `json:"security_groups,omitempty"`
}

type InterfaceListResponse struct {
	Offset     int                  `json:"offset"`
	Total      int                  `json:"total"`
	Limit      int                  `json:"limit"`
	Interfaces []*InterfaceResponse `json:"interfaces"`
}

type InterfaceFilters struct {
	InstanceID string `json:"instance_id" binding:"required,uuid"`
}

type InterfaceAdapter struct {
	BaseAdapter
	service         *services.InterfaceAdmin
	instanceService *services.InstanceAdmin
}

func NewInterfaceAdapter() *InterfaceAdapter {
	logger.Debug("Creating new Interface adapter")
	return &InterfaceAdapter{
		service:         &services.InterfaceAdmin{},
		instanceService: &services.InstanceAdmin{},
	}
}

func (a *InterfaceAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	return
}

func (a *InterfaceAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Interface list query with request: %+v", req)

	ctx := c.Request.Context()

	// 获取instance
	instanceUUID := req.Filters["instance_id"]
	if instanceUUID == "" {
		logger.Error("Instance ID is required for Interface list query")
		return nil, fmt.Errorf("instance_id is required")
	}
	instance, err := a.instanceService.GetInstanceByUUID(ctx, instanceUUID.(string))
	if err != nil {
		logger.Errorf("Failed to get instance by UUID %s: %v", instanceUUID, err)
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// 调用 service 层
	total, interfaces, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, instance)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Interfaces (total: %d)", len(interfaces), total)

	// 返回响应
	interfaceListResp := &InterfaceListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(interfaces),
	}
	interfaceList := make([]*InterfaceResponse, interfaceListResp.Limit)
	for i, iface := range interfaces {
		interfaceList[i], err = a.getInterfaceResponse(ctx, instance, iface)
		if err != nil {
			logger.Errorf("Failed to create instance response, %+v", err)
			ErrorResponse(c, http.StatusInternalServerError, "Internal error", err)
			return nil, err
		}
	}
	interfaceListResp.Interfaces = interfaceList
	logger.Debugf("List interfaces successfully, %+v", interfaceListResp)
	return interfaceListResp, nil
}

func (a *InterfaceAdapter) Get(c *gin.Context, id string) (interface{}, error) {
	logger.Debugf("Starting Interface get query with ID: %s", id)

	ctx := c.Request.Context()
	iface, err := a.service.GetInterfaceByUUID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}

	instance, err := a.instanceService.Get(ctx, iface.Instance)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return a.getInterfaceResponse(ctx, instance, iface)
}

func (a *InterfaceAdapter) getInterfaceResponse(ctx context.Context, instance *model.Instance, iface *model.Interface) (interfaceResp *InterfaceResponse, err error) {
	interfaceResp = &InterfaceResponse{
		BaseReference: &BaseReference{
			ID:   iface.UUID,
			Name: iface.Name,
		},
		AddressInfo: &AddressInfo{
			IPAddress: iface.Address.Address,
			Subnet: &SubnetResponse{
				ResourceReference: &ResourceReference{
					ID:   iface.Address.Subnet.UUID,
					Name: iface.Address.Subnet.Name,
				},
				Network: iface.Address.Subnet.Network,
				Netmask: iface.Address.Subnet.Netmask,
				Gateway: iface.Address.Subnet.Gateway,
				Vlan:    int(iface.Address.Subnet.Vlan),
				Type:    iface.Address.Subnet.Type,
			},
		},
		MacAddress: iface.MacAddr,
		IsPrimary:  iface.PrimaryIf,
		Inbound:    iface.Inbound,
		Outbound:   iface.Outbound,
	}
	if iface.PrimaryIf {
		if len(instance.FloatingIps) > 0 {
			floatingIps := make([]*FloatingIpInfo, len(instance.FloatingIps))
			for i, floatingip := range instance.FloatingIps {
				floatingIps[i] = &FloatingIpInfo{
					ResourceReference: &ResourceReference{
						ID:   floatingip.UUID,
						Name: floatingip.Name,
					},
					IpAddress:  floatingip.IPAddress,
					FipAddress: floatingip.FipAddress,
					Type:       floatingip.Type,
				}

				floatingIps[i] = &FloatingIpInfo{
					ResourceReference: &ResourceReference{
						ID:   floatingip.UUID,
						Name: floatingip.Name,
					},
					IpAddress:  floatingip.IPAddress,
					FipAddress: floatingip.FipAddress,
					Type:       floatingip.Type,
				}

				if floatingip.Subnet != nil {
					floatingIps[i].Vlan = floatingip.Subnet.Vlan
				}
				if floatingip.Group != nil {
					floatingIps[i].Group = &BaseReference{
						ID:   floatingip.Group.UUID,
						Name: floatingip.Group.Name,
					}
				}
			}
			interfaceResp.FloatingIps = floatingIps
		}
		if len(iface.SiteSubnets) > 0 {
			for _, site := range iface.SiteSubnets {
				siteInfo := &SiteSubnetInfo{
					ResourceReference: &ResourceReference{
						ID:   site.UUID,
						Name: site.Name,
					},
					Network: site.Network,
					Gateway: site.Gateway,
					Netmask: site.Netmask,
					Start:   site.Start,
					End:     site.End,
				}
				if site.Group != nil {
					siteInfo.Group = &BaseReference{
						ID:   site.Group.UUID,
						Name: site.Group.Name,
					}
				}
				siteInfo.Vlan = site.Vlan
				interfaceResp.SiteSubnets = append(interfaceResp.SiteSubnets, siteInfo)
			}
		}
		if len(iface.SecondAddresses) > 0 {
			for _, secondAddr := range iface.SecondAddresses {
				interfaceResp.SecondaryAddresses = append(interfaceResp.SecondaryAddresses, &AddressInfo{
					IPAddress: secondAddr.Address,
					Subnet: &SubnetResponse{
						ResourceReference: &ResourceReference{
							ID:   secondAddr.Subnet.UUID,
							Name: secondAddr.Subnet.Name,
						},
						Network: secondAddr.Subnet.Network,
						Netmask: secondAddr.Subnet.Netmask,
						Gateway: secondAddr.Subnet.Gateway,
						Vlan:    int(secondAddr.Subnet.Vlan),
						Type:    secondAddr.Subnet.Type,
					},
				})
			}
		}
	}
	for _, sg := range iface.SecurityGroups {
		interfaceResp.SecurityGroups = append(interfaceResp.SecurityGroups, &ResourceReference{
			ID:   sg.UUID,
			Name: sg.Name,
		})
	}
	return
}
