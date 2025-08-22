/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    auto-generated - Initial implementation
 *
 *
 * Purpose: IP Address resource adapter
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

type SubnetType string

type AddressResponse struct {
	*ResourceReference
	Address         string           `json:"address"`
	Netmask         string           `json:"netmask"`
	Type            SubnetType       `json:"type"`
	Allocated       bool             `json:"allocated"`
	Reserved        bool             `json:"reserved"`
	SubnetID        int64            `json:"subnet_id"`
	TargetInterface *TargetInterface `json:"interface"`
}

type AddressListResponse struct {
	Offset    int                `json:"offset"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Addresses []*AddressResponse `json:"addresses"`
}

type AddressFilters struct {
	SubnetID  string `json:"subnet_id" binding:"required"` // 子网ID，必传
	Address   string `json:"address,omitempty"`            // IP地址模糊匹配
	Allocated *bool  `json:"allocated,omitempty"`          // 是否已分配
	Reserved  *bool  `json:"reserved,omitempty"`           // 是否保留
	Type      string `json:"type,omitempty"`               // 子网类型
}

type AddressAdapter struct {
	BaseAdapter
	subnetService     *services.SubnetAdmin
	interfaceService  *services.InterfaceAdmin
	floatingIpService *services.FloatingIpAdmin
	instanceService   *services.InstanceAdmin
}

func NewAddressAdapter() *AddressAdapter {
	logger.Debug("Creating new Address adapter")
	return &AddressAdapter{
		subnetService:     &services.SubnetAdmin{},
		interfaceService:  &services.InterfaceAdmin{},
		floatingIpService: &services.FloatingIpAdmin{},
		instanceService:   &services.InstanceAdmin{},
	}
}

func (a *AddressAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Address filters: %+v", filtersMap)
	ctx := c.Request.Context()

	filters, err := ParseFilters[AddressFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// 子网ID查询（必传）
	subnet, err := a.subnetService.GetSubnetByUUID(ctx, filters.SubnetID)
	if err != nil {
		return
	}
	conditions = append(conditions, fmt.Sprintf("subnet_id = %d", subnet.ID))
	logger.Debugf("Added subnet_id filter: %s -> subnet_id = %d", filters.SubnetID, subnet.ID)

	// IP地址模糊匹配
	if filters.Address != "" {
		conditions = append(conditions, fmt.Sprintf("address like '%%%s%%'", filters.Address))
		logger.Debugf("Added address filter: %s", filters.Address)
	}

	// 分配状态查询
	if filters.Allocated != nil {
		conditions = append(conditions, fmt.Sprintf("allocated = %t", *filters.Allocated))
		logger.Debugf("Added allocated filter: %t", *filters.Allocated)
	}

	// 保留状态查询
	if filters.Reserved != nil {
		conditions = append(conditions, fmt.Sprintf("reserved = %t", *filters.Reserved))
		logger.Debugf("Added reserved filter: %t", *filters.Reserved)
	}

	// 类型查询
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

func (a *AddressAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Address list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, fmt.Errorf("failed to process filters: %w", err)
	}

	// 调用 service 层
	logger.Debugf("Calling service layer with query: %s", query)
	req.Order = "address::inet"
	total, addresses, err := a.subnetService.AddressList(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Addresses (total: %d)", len(addresses), total)

	addressListResp := &AddressListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(addresses),
	}
	addressListResp.Addresses = make([]*AddressResponse, addressListResp.Limit)
	for i, address := range addresses {
		addressListResp.Addresses[i], err = a.getAddressResponse(ctx, address)
		if err != nil {
			return nil, err
		}
	}
	return addressListResp, nil
}

func (a *AddressAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting Address get query with ID: %s", id)

	ctx := c.Request.Context()
	address, err := a.subnetService.GetAddressByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getAddressResponse(ctx, address)
	if err != nil {
		return
	}

	logger.Debugf("Get address successfully: %+v", resp)
	return
}

func (a *AddressAdapter) getAddressResponse(ctx context.Context, address *model.Address) (addressResp *AddressResponse, err error) {
	addressResp = &AddressResponse{
		ResourceReference: &ResourceReference{
			ID:        address.UUID,
			CreatedAt: address.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: address.UpdatedAt.Format(TimeStringForMat),
		},
		Address:   address.Address,
		Netmask:   address.Netmask,
		Type:      SubnetType(address.Type),
		Allocated: address.Allocated,
		Reserved:  address.Reserved,
		SubnetID:  address.SubnetID,
	}
	if address.Interface != 0 {
		iface := &model.Interface{}
		iface, err = a.interfaceService.Fetch(ctx, address.Interface)
		if err != nil {
			logger.Errorf("Failed to get interface for address %s, err: %v", address.Address, err)
			return
		}
		addressResp.TargetInterface = &TargetInterface{
			ResourceReference: &ResourceReference{
				ID:   iface.UUID,
				Name: iface.Name,
			},
			MacAddr: iface.MacAddr,
		}
		if address.Subnet.Type == "internal" {
			if iface.Instance > 0 {
				instance := &model.Instance{}
				instance, err = a.instanceService.Fetch(ctx, iface.Instance)
				if err != nil {
					logger.Errorf("Failed to get instance for interface %d, err: %v", iface.Instance, err)
					return
				}
				addressResp.TargetInterface.FromInstance = &InstanceInfo{
					ResourceReference: &ResourceReference{
						ID: instance.UUID,
					},
					Hostname: instance.Hostname,
				}
				if instance.OwnerInfo != nil {
					addressResp.TargetInterface.FromInstance.ResourceReference.Owner = instance.OwnerInfo.Name
				}
			}
		} else {
			floatingIp := &model.FloatingIp{}
			floatingIp, err = a.floatingIpService.GetFloatingIpByAddress(ctx, address.Address)
			if err != nil {
				logger.Errorf("Failed to get floating IP for interface %d, err: %v", iface.ID, err)
				return addressResp, nil
			}
			if err == nil && floatingIp.Instance != nil {
				addressResp.TargetInterface.FromInstance = &InstanceInfo{
					ResourceReference: &ResourceReference{
						ID: floatingIp.Instance.UUID,
					},
					Hostname: floatingIp.Instance.Hostname,
				}
				if floatingIp.OwnerInfo != nil {
					addressResp.TargetInterface.FromInstance.ResourceReference.Owner = floatingIp.OwnerInfo.Name
				}
			}
		}
	}

	return
}
