/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"context"
	"fmt"
	"strings"
	"web/src/model"

	"github.com/jinzhu/gorm"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

type InstanceAdmin struct{}

type ExecutionCommand struct {
	Control string
	Command string
}

type NetworkLink struct {
	MacAddr string `json:"ethernet_mac_address"`
	Mtu     uint   `json:"mtu"`
	ID      string `json:"id"`
	Type    string `json:"type,omitempty"`
}

type VolumeInfo struct {
	ID      int64  `json:"id"`
	UUID    string `json:"uuid"`
	Device  string `json:"device"`
	Booting bool   `json:"booting"`
}

type InstanceNetwork struct {
	Type    string          `json:"type,omitempty"`
	Address string          `json:"ip_address"`
	Netmask string          `json:"netmask"`
	Link    string          `json:"link"`
	ID      string          `json:"id"`
	Routes  []*NetworkRoute `json:"routes,omitempty"`
}

type InstanceData struct {
	Userdata   string             `json:"userdata"`
	DNS        string             `json:"dns"`
	Vlans      []*VlanInfo        `json:"vlans"`
	Networks   []*InstanceNetwork `json:"networks"`
	Links      []*NetworkLink     `json:"links"`
	Volumes    []*VolumeInfo      `json:"volumes"`
	Keys       []string           `json:"keys"`
	RootPasswd string             `json:"root_passwd"`
	LoginPort  int                `json:"login_port"`
	OSCode     string             `json:"os_code"`
}

type InstancesData struct {
	Instances []*model.Instance `json:"instancedata"`
	IsAdmin   bool              `json:"is_admin"`
}

func (a *InstanceAdmin) GetHyperGroup(ctx context.Context, zoneID int64, skipHyper int32) (hyperGroup string, err error) {
	ctx, db := GetContextDB(ctx)
	hypers := []*model.Hyper{}
	where := fmt.Sprintf("zone_id = %d and status = 1 and hostid <> %d", zoneID, skipHyper)
	if err = db.Where(where).Find(&hypers).Error; err != nil {
		logger.Error("Hypers query failed", err)
		return "", NewCLError(ErrSQLSyntaxError, "Failed to query hypervisors", err)
	}
	if len(hypers) == 0 {
		logger.Error("No qualified hypervisor")
		return "", NewCLError(ErrNoQualifiedHypervisor, "No qualified hypervisor found", nil)
	}
	hyperGroup = fmt.Sprintf("group-zone-%d", zoneID)
	for i, h := range hypers {
		if i == 0 {
			hyperGroup = fmt.Sprintf("%s:%d", hyperGroup, h.Hostid)
		} else {
			hyperGroup = fmt.Sprintf("%s,%d", hyperGroup, h.Hostid)
		}
	}
	return
}

func (a *InstanceAdmin) Get(ctx context.Context, id int64) (instance *model.Instance, err error) {
	if id <= 0 {
		err = NewCLError(ErrInvalidParameter, "Invalid instance ID", nil)
		logger.Error(err)
		return
	}
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	instance = &model.Instance{Model: model.Model{ID: id}}
	if err = db.Preload("Volumes").Preload("Image").Preload("Zone").Preload("Flavor").Preload("Keys").Where(where).Take(instance).Error; err != nil {
		logger.Errorf("Failed to query instance, %v", err)
		return nil, NewCLError(ErrInstanceNotFound, "Instance not found", err)
	}

	if err = db.Preload("Group").Preload("Subnet").Where("instance_id = ?", instance.ID).Order("updated_at").Find(&instance.FloatingIps).Error; err != nil {
		logger.Errorf("Failed to query floating ip(s), %v", err)
		return nil, NewCLError(ErrSQLSyntaxError, "Failed to query floating ip(s) for instance", err)
	}
	if err = db.Preload("SiteSubnets").Preload("SiteSubnets.Group").Preload("SecurityGroups").Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("addresses.updated_at")
	}).Preload("SecondAddresses.Subnet").Where("instance = ?", instance.ID).Find(&instance.Interfaces).Error; err != nil {
		logger.Errorf("Failed to query interfaces %v", err)
		return nil, NewCLError(ErrSQLSyntaxError, "Failed to query interfaces for instance", err)
	}
	permit := memberShip.ValidateOwner(model.Reader, instance.Owner)
	if !permit {
		logger.Error("Not authorized to read the instance")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the instance", nil)
		return
	}
	permit = memberShip.CheckPermission(model.Admin)
	if permit {
		instance.OwnerInfo = &model.Organization{Model: model.Model{ID: instance.Owner}}
		if err = db.Take(instance.OwnerInfo).Error; err != nil {
			logger.Error("Failed to query owner info", err)
			return nil, NewCLError(ErrOwnerNotFound, "Failed to query owner info for instance", err)
		}
	}

	return
}

func (a *InstanceAdmin) Fetch(ctx context.Context, id int64) (instance *model.Instance, err error) {
	if id <= 0 {
		err = NewCLError(ErrInvalidParameter, "Invalid instance ID", nil)
		logger.Error(err)
		return
	}
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	instance = &model.Instance{Model: model.Model{ID: id}}
	if err = db.Where(where).Take(instance).Error; err != nil {
		logger.Errorf("Failed to query instance, %v", err)
		return nil, NewCLError(ErrInstanceNotFound, "Instance not found", err)
	}

	permit := memberShip.ValidateOwner(model.Reader, instance.Owner)
	if !permit {
		logger.Error("Not authorized to read the instance")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the instance", nil)
		return
	}
	permit = memberShip.CheckPermission(model.Admin)
	if permit {
		instance.OwnerInfo = &model.Organization{Model: model.Model{ID: instance.Owner}}
		if err = db.Take(instance.OwnerInfo).Error; err != nil {
			logger.Error("Failed to query owner info", err)
			return nil, NewCLError(ErrOwnerNotFound, "Failed to query owner info for instance", err)
		}
	}

	return
}

func (a *InstanceAdmin) GetInstanceByUUID(ctx context.Context, uuID string) (instance *model.Instance, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	where := memberShip.GetWhere()
	instance = &model.Instance{}
	if err = db.Preload("Volumes").Preload("Image").Preload("Zone").Preload("Flavor").Preload("Keys").Where(where).Where("uuid = ?", uuID).Take(instance).Error; err != nil {
		logger.Errorf("Failed to query instance, %v", err)
		return nil, NewCLError(ErrInstanceNotFound, "Instance not found", err)
	}

	if err = db.Preload("Group").Preload("Subnet").Where("instance_id = ?", instance.ID).Order("updated_at").Find(&instance.FloatingIps).Error; err != nil {
		logger.Errorf("Failed to query floating ip(s), %v", err)
		return nil, NewCLError(ErrSQLSyntaxError, "Failed to query floating ip(s) for instance", err)
	}
	if err = db.Preload("SiteSubnets").Preload("SiteSubnets.Group").Preload("SecurityGroups").Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
		return db.Order("addresses.updated_at")
	}).Preload("SecondAddresses.Subnet").Where("instance = ?", instance.ID).Find(&instance.Interfaces).Error; err != nil {
		logger.Errorf("Failed to query interfaces %v", err)
		return nil, NewCLError(ErrSQLSyntaxError, "Failed to query interfaces for instance", err)
	}
	if instance.RouterID > 0 {
		instance.Router = &model.Router{Model: model.Model{ID: instance.RouterID}}
		if err = db.Take(instance.Router).Error; err != nil {
			logger.Errorf("Failed to query router, %v", err)
			return nil, NewCLError(ErrRouterNotFound, "Failed to query router for instance", err)
		}
	}
	permit := memberShip.ValidateOwner(model.Reader, instance.Owner)
	if !permit {
		logger.Error("Not authorized to read the instance")
		err = NewCLError(ErrPermissionDenied, "Not authorized to read the instance", nil)
		return
	}
	return
}

func (a *InstanceAdmin) List(ctx context.Context, offset, limit int64, order, query string) (total int64, instances []*model.Instance, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		return
	}
	db := DB()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}
	logger.Debugf("The query in admin console is %s", query)

	where := memberShip.GetWhere()
	instances = []*model.Instance{}
	if err = db.Model(&model.Instance{}).Where(where).Where(query).Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to count instance(s)", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Volumes").Preload("Image").Preload("Zone").Preload("Flavor").Preload("Keys").Where(where).Where(query).Find(&instances).Error; err != nil {
		logger.Errorf("Failed to query instance(s), %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to query instance(s)", err)
		return
	}
	db = db.Offset(0).Limit(-1)
	for _, instance := range instances {
		if err = db.Preload("SiteSubnets").Preload("SiteSubnets.Group").Preload("SecurityGroups").Preload("Address").Preload("Address.Subnet").Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
			return db.Order("addresses.updated_at")
		}).Preload("SecondAddresses.Subnet").Where("instance = ?", instance.ID).Find(&instance.Interfaces).Error; err != nil {
			logger.Errorf("Failed to query interfaces %v", err)
			err = NewCLError(ErrSQLSyntaxError, "Failed to query interfaces", err)
			return
		}

		if err = db.Preload("Group").Preload("Subnet").Order("updated_at").Where("instance_id = ?", instance.ID).Find(&instance.FloatingIps).Error; err != nil {
			logger.Errorf("Failed to query floating ip(s), %v", err)
			err = NewCLError(ErrSQLSyntaxError, "Failed to query floating ip(s) for instance", err)
			return
		}

		if instance.RouterID > 0 {
			instance.Router = &model.Router{Model: model.Model{ID: instance.RouterID}}
			if err = db.Take(instance.Router).Error; err != nil {
				logger.Errorf("Failed to query floating ip(s), %v", err)
				err = NewCLError(ErrRouterNotFound, "Failed to query router for instance", err)
				return
			}
		}
		permit := memberShip.CheckPermission(model.Admin)
		if permit {
			instance.OwnerInfo = &model.Organization{Model: model.Model{ID: instance.Owner}}
			if err = db.Take(instance.OwnerInfo).Error; err != nil {
				logger.Error("Failed to query owner info", err)
				err = NewCLError(ErrOwnerNotFound, "Failed to query owner info for instance", err)
				return
			}
		}
	}

	return
}

func (a *InstanceAdmin) BaseList(ctx context.Context, offset, limit int64, order, query string) (total int64, instances []*model.Instance, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		return
	}
	db := DB()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}
	logger.Debugf("The query in admin console is %s", query)

	where := memberShip.GetWhere()
	instances = []*model.Instance{}
	if err = db.Model(&model.Instance{}).Where(where).Where(query).Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to count instance(s)", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Where(where).Where(query).Find(&instances).Error; err != nil {
		logger.Errorf("Failed to query instance(s), %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to query instance(s)", err)
		return
	}
	return
}

func (a *InstanceAdmin) GetInstanceCount(ctx context.Context, query string) (count int64, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		return
	}

	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	if err = db.Model(&model.Instance{}).Where(where).Where(query).Count(&count).Error; err != nil {
		logger.Errorf("Failed to count instances, %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count instance(s)", err)
		return
	}

	return
}

func (a *InstanceAdmin) GetInstanceCountByHyper(ctx context.Context, hostID int32) (count int64, err error) {
	query := fmt.Sprintf("hyper=%d", hostID)
	return a.GetInstanceCount(ctx, query)
}

func (a *InstanceAdmin) GetInstanceCountByZone(ctx context.Context, zoneID int64) (count int64, err error) {
	query := fmt.Sprintf("zone_id=%d", zoneID)
	return a.GetInstanceCount(ctx, query)
}

func (a *InstanceAdmin) GetInstanceCountsByHypers(ctx context.Context, hostIDs []int32) (countsByHyper map[int32]int64, err error) {
	if len(hostIDs) == 0 {
		return make(map[int32]int64), nil
	}

	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
		return nil, NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
	}

	_, db := GetContextDB(ctx)

	var results []struct {
		Hyper int32 `gorm:"column:hyper"`
		Count int64 `gorm:"column:count"`
	}

	if err = db.Model(&model.Instance{}).
		Select("hyper, COUNT(*) as count").
		Where("hyper IN (?)", hostIDs).
		Group("hyper").
		Scan(&results).Error; err != nil {
		logger.Errorf("Failed to batch count instances by hyper IDs: %v", err)
		return nil, NewCLError(ErrSQLSyntaxError, "Failed to count instances", err)
	}

	countsByHyper = make(map[int32]int64)
	for _, hostID := range hostIDs {
		countsByHyper[hostID] = 0
	}

	for _, result := range results {
		countsByHyper[result.Hyper] = result.Count
	}

	return countsByHyper, nil
}

func (a *InstanceAdmin) ListWithJoins(ctx context.Context, offset, limit int64, order, query string) (total int64, instances []*model.Instance, err error) {
	memberShip := GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		logger.Error("Not authorized for this operation")
		err = NewCLError(ErrPermissionDenied, "Not authorized for this operation", nil)
		return
	}

	db := DB()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "instances.created_at DESC"
	}

	logger.Debugf("The query in admin console is %s", query)

	where := ""
	if memberShip.OrgName == "admin" && memberShip.Role == model.Admin {
		where = ""
	} else {
		where = fmt.Sprintf("instances.owner = %d", memberShip.OrgID)
	}
	instances = []*model.Instance{}

	// 构建基础JOIN查询
	joinDB := db.Model(&model.Instance{}).
		Joins("LEFT JOIN organizations ON organizations.id = instances.owner").
		Joins("LEFT JOIN interfaces ON interfaces.instance = instances.id AND interfaces.deleted_at IS NULL").
		Joins("LEFT JOIN addresses ON addresses.interface = interfaces.id AND addresses.deleted_at IS NULL").
		Joins("LEFT JOIN addresses AS addresses2 ON addresses2.second_interface = interfaces.id AND addresses2.deleted_at IS NULL").
		Joins("LEFT JOIN floating_ips ON floating_ips.instance_id = instances.id AND floating_ips.deleted_at IS NULL").
		Joins("LEFT JOIN secgroup_ifaces ON secgroup_ifaces.interface_id = interfaces.id").
		Joins("LEFT JOIN security_groups ON security_groups.id = secgroup_ifaces.security_group_id AND security_groups.deleted_at IS NULL").
		Joins("LEFT JOIN subnets AS site_subnets ON site_subnets.interface = interfaces.id AND site_subnets.deleted_at IS NULL").
		Joins("LEFT JOIN addresses AS site_addresses ON site_addresses.subnet_id = site_subnets.id AND site_addresses.deleted_at IS NULL")

	if strings.Contains(query, "adjust_rule_group") {
		joinDB = joinDB.
			Joins("LEFT JOIN vm_rule_links AS adjust_rule_links ON adjust_rule_links.instance_id = instances.uuid AND adjust_rule_links.deleted_at IS NULL").
			Joins("LEFT JOIN adjust_rule_group ON adjust_rule_group.uuid = adjust_rule_links.group_uuid AND adjust_rule_group.deleted_at IS NULL")
	}

	// 计数
	if err = joinDB.Model(&model.Instance{}).Where(where).Where(query).Select("COUNT(DISTINCT instances.id)").Count(&total).Error; err != nil {
		logger.Errorf("Failed to count instances with joins: %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count instance(s)", err)
		return
	}

	// 应用排序、分页和限制
	joinDB = dbs.Sortby(joinDB.Offset(offset).Limit(limit), order)

	// 查询实例
	if err = joinDB.
		Preload("Volumes").
		Preload("Image").
		Preload("Zone").
		Preload("Flavor").
		Preload("Router").
		Preload("Interfaces", func(db *gorm.DB) *gorm.DB {
			return db.
				Preload("SiteSubnets").
				Preload("SiteSubnets.Group").
				Preload("SecurityGroups").
				Preload("Address").
				Preload("Address.Subnet").
				Preload("SecondAddresses", func(db *gorm.DB) *gorm.DB {
					return db.Order("addresses.updated_at")
				}).
				Preload("SecondAddresses.Subnet")
		}).
		Preload("FloatingIps", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Group").Preload("Subnet").Order("updated_at")
		}).
		Where(where).
		Where(query).
		Group("instances.id").Find(&instances).Error; err != nil {
		logger.Errorf("Failed to query instances with joins: %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to query instance(s)", err)
		return
	}

	if err = a.batchLoadOwnerInfo(ctx, instances); err != nil {
		logger.Errorf("Failed to load owner info: %v", err)
		return
	}

	return
}

func (a *InstanceAdmin) batchLoadOwnerInfo(ctx context.Context, instances []*model.Instance) error {
	// 收集所有需要查询的组织ID
	ownerIDs := make(map[int64]bool)
	for _, instance := range instances {
		ownerIDs[instance.Owner] = true
	}

	if len(ownerIDs) == 0 {
		return nil
	}

	// 批量查询组织信息
	var orgIDs []int64
	for id := range ownerIDs {
		orgIDs = append(orgIDs, id)
	}

	var orgs []*model.Organization
	if err := DB().Where("id IN (?)", orgIDs).Find(&orgs).Error; err != nil {
		err = NewCLError(ErrOwnerNotFound, "Failed to query owner info for instance", err)
		return err
	}

	// 创建组织ID到信息的映射
	orgMap := make(map[int64]*model.Organization)
	for _, org := range orgs {
		orgMap[org.ID] = org
	}

	// 为每个实例设置组织信息
	for _, instance := range instances {
		if org, exists := orgMap[instance.Owner]; exists {
			instance.OwnerInfo = org
		}
	}

	return nil
}

func (a *InstanceAdmin) Count(ctx context.Context) (count int64, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	if err = db.Model(&model.Instance{}).Where(where).Count(&count).Error; err != nil {
		logger.Error("Failed to count instances", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count instances", err)
	}
	return
}

func (a *InstanceAdmin) SumCPU(ctx context.Context) (total int64, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	var result struct {
		Total int64
	}
	if err = db.Model(&model.Instance{}).Select("SUM(cpu) as total").Where(where).Scan(&result).Error; err != nil {
		logger.Error("Failed to sum CPU", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to sum CPU", err)
		return
	}
	return result.Total, nil
}

func (a *InstanceAdmin) SumMemory(ctx context.Context) (total int64, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	var result struct {
		Total int64
	}
	if err = db.Model(&model.Instance{}).Select("SUM(memory) as total").Where(where).Scan(&result).Error; err != nil {
		logger.Error("Failed to sum memory", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to sum memory", err)
		return
	}
	return result.Total, nil
}
