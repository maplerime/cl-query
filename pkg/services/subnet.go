/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package services

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"web/src/model"

	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/dbs"
)

var (
	subnetAdmin = &SubnetAdmin{}
	vniMax      = 16777215
	vniMin      = 4096
)

type SubnetAdmin struct{}

type SubnetStats struct {
	IPCount        int64
	AllocatedCount int64
	ReservedCount  int64
	IdleCount      int64
}

type SubnetWithStats struct {
	Subnet *model.Subnet
	Stats  *SubnetStats
}

type subnetStatRow struct {
	ID             int64 `gorm:"column:id"`
	IPCount        int64 `gorm:"column:stat_ip_count"`
	AllocatedCount int64 `gorm:"column:stat_allocated_count"`
	ReservedCount  int64 `gorm:"column:stat_reserved_count"`
	IdleCount      int64 `gorm:"column:stat_idle_count"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
	return
}

func getValidVni(ctx context.Context) (vni int, err error) {
	ctx, db := GetContextDB(ctx)
	count := 1
	for count > 0 {
		vni = rand.Intn(vniMax-vniMin) + vniMin
		if err = db.Model(&model.Subnet{}).Where("vlan = ?", vni).Count(&count).Error; err != nil {
			logger.Error("Failed to query existing vlan, %v", err)
			return
		}
	}
	return
}

func (a *SubnetAdmin) Get(ctx context.Context, id int64) (subnet *model.Subnet, err error) {
	if id <= 0 {
		err = NewCLError(ErrInvalidParameter, "Invalid subnet ID", err)
		logger.Error(err)
		return
	}
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	subnet = &model.Subnet{Model: model.Model{ID: id}}
	err = db.Preload("Router").Preload("Group").Take(subnet).Error
	if err != nil {
		logger.Error("DB failed to query subnet ", err)
		err = NewCLError(ErrSubnetNotFound, "Subnet not found", err)
		return
	}
	if subnet.RouterID > 0 {
		subnet.Router = &model.Router{Model: model.Model{ID: subnet.RouterID}}
		err = db.Take(subnet.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			err = NewCLError(ErrRouterNotFound, "Router not found", err)
			return
		}
	}
	if subnet.Type == "internal" {
		permit := memberShip.ValidateOwner(model.Reader, subnet.Owner)
		if !permit {
			logger.Error("Not authorized to read the subnet")
			err = NewCLError(ErrPermissionDenied, "Not authorized to read the subnet", nil)
			return
		}
	}
	return
}

func (a *SubnetAdmin) GetSubnetByUUID(ctx context.Context, uuID string) (subnet *model.Subnet, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	subnet = &model.Subnet{}
	err = db.Preload("Router").Preload("Group").Where("uuid = ?", uuID).Take(subnet).Error
	if err != nil {
		logger.Error("Failed to query subnet, %v", err)
		err = NewCLError(ErrSubnetNotFound, "Subnet not found", err)
		return
	}
	if subnet.RouterID > 0 {
		subnet.Router = &model.Router{Model: model.Model{ID: subnet.RouterID}}
		err = db.Take(subnet.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			err = NewCLError(ErrRouterNotFound, "Router not found", err)
			return
		}
	}
	if subnet.Type == "internal" {
		permit := memberShip.ValidateOwner(model.Reader, subnet.Owner)
		if !permit {
			logger.Error("Not authorized to read the subnet")
			err = NewCLError(ErrPermissionDenied, "Not authorized to read the subnet", nil)
			return
		}
	}
	return
}

func (a *SubnetAdmin) GetSubnetByName(ctx context.Context, name string) (subnet *model.Subnet, err error) {
	ctx, db := GetContextDB(ctx)
	memberShip := GetMemberShip(ctx)
	subnet = &model.Subnet{}
	err = db.Preload("Router").Preload("Group").Where("name = ?", name).Take(subnet).Error
	if err != nil {
		logger.Error("Failed to query subnet ", err)
		err = NewCLError(ErrSubnetNotFound, "Subnet not found", err)
		return
	}
	if subnet.RouterID > 0 {
		subnet.Router = &model.Router{Model: model.Model{ID: subnet.RouterID}}
		err = db.Take(subnet.Router).Error
		if err != nil {
			logger.Error("Failed to query router ", err)
			err = NewCLError(ErrRouterNotFound, "Router not found", err)
			return
		}
	}
	if subnet.Type == "internal" {
		permit := memberShip.ValidateOwner(model.Reader, subnet.Owner)
		if !permit {
			logger.Error("Not authorized to read the subnet")
			err = NewCLError(ErrPermissionDenied, "Not authorized to read the subnet", nil)
			return
		}
	}
	return
}

func (a *SubnetAdmin) GetSubnet(ctx context.Context, reference *BaseReference) (subnet *model.Subnet, err error) {
	if reference == nil || (reference.ID == "" && reference.Name == "") {
		err = NewCLError(ErrInvalidParameter, "Subnet base reference must be provided with either uuid or name", nil)
		return
	}
	if reference.ID != "" {
		subnet, err = a.GetSubnetByUUID(ctx, reference.ID)
		return
	}
	if reference.Name != "" {
		subnet, err = a.GetSubnetByName(ctx, reference.Name)
		return
	}
	return
}

func (a *SubnetAdmin) CountIdleAddressesForSubnet(ctx context.Context, subnet *model.Subnet) (int64, error) {
	ctx, db := GetContextDB(ctx)
	var idleCount int64

	err := db.Model(&model.Address{}).
		Where("subnet_id = ?", subnet.ID).
		Where("allocated = ?", "f").
		Where("reserved = ?", "f").
		Where("address != ?", subnet.Gateway).
		Count(&idleCount).Error

	if err != nil {
		if err.Error() != "record not found" {
			return 0, NewCLError(ErrSQLSyntaxError, "Failed to count idle addresses", err)
		}
	}

	return idleCount, nil
}

func (a *SubnetAdmin) List(ctx context.Context, offset, limit int64, order, query string, hasIdleIP bool) (total int64, subnets []*SubnetWithStats, err error) {
	ctx, db := GetContextDB(ctx)
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "created_at"
	}

	m := GetMemberShip(ctx)
	where := ""
	if m.OrgName == "admin" && m.Role == model.Admin {
		where = ""
	} else {
		where = fmt.Sprintf("subnets.owner = %d", m.OrgID)
	}
	subnets = []*SubnetWithStats{}

	// 别名带 stat_ 前缀，避免与 subnets.* 带出的列撞名；adapters 的 subnetOrderMap 引用这些别名。
	// 开头的 subnets.* 不能删：ORDER BY 优先匹配 SELECT 的输出列名，否则 created_at 这类
	// 两表同名的列会歧义（默认排序就是 -created_at）。
	statsSelect := "subnets.*, " +
		"COUNT(a.id) AS stat_ip_count, " +
		"COALESCE(SUM(CASE WHEN a.allocated = 't' AND (a.interface != 0 or a.second_interface != 0) THEN 1 ELSE 0 END), 0) AS stat_allocated_count, " +
		"COALESCE(SUM(CASE WHEN a.reserved = 't' THEN 1 ELSE 0 END), 0) AS stat_reserved_count, " +
		"COALESCE(SUM(CASE WHEN a.allocated = 'f' AND a.reserved = 'f' THEN 1 ELSE 0 END), 0) AS stat_idle_count"

	baseQuery := db.Model(&model.Subnet{}).
		Joins("LEFT JOIN addresses a ON a.subnet_id = subnets.id AND a.deleted_at IS NULL AND a.address != COALESCE(subnets.gateway, '')").
		Where(where).
		Where(query).
		Group("subnets.id").
		Select(statsSelect)

	if hasIdleIP {
		baseQuery = baseQuery.Where("EXISTS (SELECT 1 FROM addresses a3 WHERE a3.subnet_id = subnets.id AND a3.deleted_at IS NULL AND a3.address != COALESCE(subnets.gateway, '') AND a3.allocated = 'f' AND a3.reserved = 'f')")
	}

	if err = baseQuery.Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Database failed to count subnets", err)
		return
	}

	// 第一段：聚合出 ID 与四个计数
	var rows []subnetStatRow
	statQuery := dbs.Sortby(baseQuery.Offset(offset).Limit(limit), order)
	if err = statQuery.Scan(&rows).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Database failed to query subnets", err)
		return
	}
	if len(rows) == 0 {
		return
	}

	// 第二段：按 ID 装配完整对象
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	records := []*model.Subnet{}
	if err = db.Preload("Group").Preload("Router").Where("id IN (?)", ids).Find(&records).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Database failed to query subnets", err)
		return
	}
	recordMap := make(map[int64]*model.Subnet, len(records))
	for _, record := range records {
		recordMap[record.ID] = record
	}

	// IN 查询不保证顺序，按第一段的顺序回排
	for _, row := range rows {
		record, ok := recordMap[row.ID]
		if !ok {
			continue
		}
		subnets = append(subnets, &SubnetWithStats{
			Subnet: record,
			Stats: &SubnetStats{
				IPCount:        row.IPCount,
				AllocatedCount: row.AllocatedCount,
				ReservedCount:  row.ReservedCount,
				IdleCount:      row.IdleCount,
			},
		})
	}
	return
}

func (a *SubnetAdmin) AddressList(ctx context.Context, offset, limit int64, order, query string) (total int64, addresses []*model.Address, err error) {
	db := DB()
	if limit == 0 {
		limit = 16
	}

	if order == "" {
		order = "address::inet"
	}

	addresses = []*model.Address{}
	if err = db.Model(&model.Address{}).Where(query).Count(&total).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to count addresses", err)
		return
	}
	db = dbs.Sortby(db.Offset(offset).Limit(limit), order)
	if err = db.Preload("Subnet").Where(query).Find(&addresses).Error; err != nil {
		err = NewCLError(ErrSQLSyntaxError, "Failed to query addresses", err)
		return
	}
	return
}

func (a *SubnetAdmin) GetAddressByUUID(ctx context.Context, uuID string) (address *model.Address, err error) {
	ctx, db := GetContextDB(ctx)
	address = &model.Address{}
	err = db.Preload("Subnet").Where("uuid = ?", uuID).Take(address).Error
	if err != nil {
		logger.Error("Failed to query address, %v", err)
		err = NewCLError(ErrAddressNotFound, "Address not found", err)
		return
	}
	return
}

func (a *SubnetAdmin) GetAddressesBySubnet(ctx context.Context, subnetID int64) (addresses []*model.Address, err error) {
	ctx, db := GetContextDB(ctx)
	addresses = []*model.Address{}
	err = db.Where("subnet_id = ?", subnetID).Order("address::inet").Find(&addresses).Error
	if err != nil {
		logger.Error("Failed to query addresses by subnet_id, %v", err)
		err = NewCLError(ErrSQLSyntaxError, "Find to query addresses", err)
		return
	}
	return
}

func (a *SubnetAdmin) AddressStatistics(ctx context.Context, subnet *model.Subnet) (total, allocated, reserved, idle int64, err error) {
	ctx, db := GetContextDB(ctx)
	query := db.Model(&model.Address{}).
		Select(`
			COUNT(*) as total,
			SUM(CASE WHEN allocated = 't' AND (interface != 0 or second_interface != 0) THEN 1 ELSE 0 END) as allocated,
			SUM(CASE WHEN reserved = 't' THEN 1 ELSE 0 END) as reserved,
			SUM(CASE WHEN allocated = 'f' AND reserved = 'f' THEN 1 ELSE 0 END) as idle
		`).
		Where("subnet_id = ? AND address != ?", subnet.ID, subnet.Gateway)

	var result struct {
		Total     int64
		Allocated int64
		Reserved  int64
		Idle      int64
	}

	if err = query.Scan(&result).Error; err != nil {
		logger.Error("Failed to count addresses for subnet", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count addresses for subnet", err)
		return
	}

	return result.Total, result.Allocated, result.Reserved, result.Idle, nil
}

func (a *SubnetAdmin) Count(ctx context.Context) (count int64, err error) {
	memberShip := GetMemberShip(ctx)
	ctx, db := GetContextDB(ctx)
	where := memberShip.GetWhere()

	if err = db.Model(&model.Subnet{}).Where(where).Count(&count).Error; err != nil {
		logger.Error("Failed to count subnets", err)
		err = NewCLError(ErrSQLSyntaxError, "Failed to count subnets", err)
	}
	return
}
