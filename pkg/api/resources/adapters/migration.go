/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2026 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Initial implementation
 *
 *
 * Purpose: Migration resource adapter
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

// MigrationPhaseResponse 迁移阶段（对应 cloudland model.Task，tasks.mission = migrations.id）
type MigrationPhaseResponse struct {
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// MigrationResponse 迁移任务响应：实例、源/目标节点信息平铺，供上层服务直接使用
type MigrationResponse struct {
	*ResourceReference
	InstanceID       string                    `json:"instance_id"`
	InstanceHostname string                    `json:"instance_hostname"`
	SourceHyper      int32                     `json:"source_hyper"`
	SourceHyperName  string                    `json:"source_hyper_name"`
	TargetHyper      int32                     `json:"target_hyper"`
	TargetHyperName  string                    `json:"target_hyper_name"`
	Force            bool                      `json:"force"`
	Type             string                    `json:"type"`
	Status           string                    `json:"status"`
	Phases           []*MigrationPhaseResponse `json:"phases"`
}

// MigrationListResponse 迁移任务列表响应
type MigrationListResponse struct {
	Offset     int                  `json:"offset"`
	Total      int                  `json:"total"`
	Limit      int                  `json:"limit"`
	Migrations []*MigrationResponse `json:"migrations"`
}

// MigrationFilters 迁移任务查询过滤参数
type MigrationFilters struct {
	UUIDs    []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	Hostname string   `json:"hostname,omitempty"` // 按 instance hostname 模糊过滤（子查询 instances 表）
	Status   string   `json:"status,omitempty"`   // 按迁移任务状态精确过滤
}

// MigrationAdapter migration 资源适配器（resource type = "migration"）
type MigrationAdapter struct {
	BaseAdapter
	service *services.MigrationAdmin
}

// NewMigrationAdapter 创建 migration adapter
func NewMigrationAdapter() *MigrationAdapter {
	logger.Debug("Creating new Migration adapter")
	return &MigrationAdapter{
		service: &services.MigrationAdmin{},
	}
}

// CheckPermission migration 为管理员级资源，覆盖 BaseAdapter 的 Reader 校验，
// 对齐 services/migration.go 中 GetMigrationByUUID 等方法的 model.Admin 语义
func (a *MigrationAdapter) CheckPermission(ctx context.Context) error {
	memberShip := GetMemberShip(ctx)
	// 权限不足则拒绝
	if !memberShip.CheckPermission(model.Admin) {
		logger.Error("Not authorized to access migrations")
		return NewCLError(ErrPermissionDenied, "Not authorized to access migrations", nil)
	}
	return nil
}

// MakeQuery 根据过滤参数组装 WHERE 片段。
// 支持：instance hostname（子查询 instances 表）、status 精确匹配、uuids 列表
func (a *MigrationAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (query string, err error) {
	logger.Debugf("Processing Migration filters: %+v", filtersMap)

	filters, err := ParseFilters[MigrationFilters](c)
	if err != nil {
		return
	}

	var conditions []string

	// instance hostname 过滤：migrations 表无 hostname，经 instances 表子查询关联
	if filters.Hostname != "" {
		conditions = append(conditions, fmt.Sprintf(
			"instance_id IN (SELECT id FROM instances WHERE hostname LIKE '%%%s%%' AND deleted_at IS NULL)",
			filters.Hostname))
		logger.Debugf("Added instance hostname filter: %s", filters.Hostname)
	}

	// status 精确过滤
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = '%s'", filters.Status))
		logger.Debugf("Added status filter: %s", filters.Status)
	}

	// UUIDs 过滤
	if filters.UUIDs != nil {
		if len(filters.UUIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
			logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
		} else {
			// 空列表表示无匹配
			conditions = append(conditions, "1=0")
		}
	}

	if len(conditions) > 0 {
		query = strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
	}

	return
}

// List 分页查询迁移任务列表
func (a *MigrationAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Migration list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	// 调用 service 层（List 已 Preload Instance/Phases）
	total, migrations, err := a.service.List(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d migrations (total: %d)", len(migrations), total)

	// 组装响应
	migrationListResp := &MigrationListResponse{
		Total:  int(total),
		Offset: req.Offset,
		Limit:  len(migrations),
	}
	migrationListResp.Migrations = make([]*MigrationResponse, migrationListResp.Limit)
	for i, migration := range migrations {
		migrationListResp.Migrations[i], err = a.getMigrationResponse(ctx, migration)
		if err != nil {
			return nil, err
		}
	}

	logger.Debugf("List migrations successfully: total=%d", migrationListResp.Total)
	return migrationListResp, nil
}

// Get 按 UUID 查询单条迁移任务
func (a *MigrationAdapter) Get(c *gin.Context, id string) (resp interface{}, err error) {
	logger.Debugf("Starting Migration get query with ID: %s", id)

	ctx := c.Request.Context()
	// GetMigrationByUUID 内部已做 Admin 权限校验并 Preload Instance/Phases
	migration, err := a.service.GetMigrationByUUID(ctx, id)
	if err != nil {
		return
	}

	resp, err = a.getMigrationResponse(ctx, migration)
	if err != nil {
		return
	}

	return
}

// getMigrationResponse 组装单条迁移任务响应：平铺 instance、源/目标 hyper 信息，附 phases
func (a *MigrationAdapter) getMigrationResponse(ctx context.Context, migration *model.Migration) (migrationResp *MigrationResponse, err error) {
	migrationResp = &MigrationResponse{
		ResourceReference: &ResourceReference{
			ID:        migration.UUID,
			Name:      migration.Name,
			CreatedAt: migration.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: migration.UpdatedAt.Format(TimeStringForMat),
		},
		SourceHyper: migration.SourceHyper,
		TargetHyper: migration.TargetHyper,
		Force:       migration.Force,
		Type:        migration.Type,
		Status:      migration.Status,
	}

	// instance 信息平铺（关联缺失时保持空值）
	if migration.Instance != nil {
		migrationResp.InstanceID = migration.Instance.UUID
		migrationResp.InstanceHostname = migration.Instance.Hostname
	}

	// 源/目标 hyper 名称：SourceHyper/TargetHyper 存的是 hypers.hostid，按 hostid 批量查询。
	// hostid = -1 表示目标未调度/未指定（hypers 表中 -1 是控制节点，必须排除，否则会误显示为控制节点名）
	hostids := make([]int32, 0, 2)
	if migration.SourceHyper >= 0 {
		hostids = append(hostids, migration.SourceHyper)
	}
	if migration.TargetHyper >= 0 {
		hostids = append(hostids, migration.TargetHyper)
	}
	if len(hostids) > 0 {
		var hypers map[int32]*model.Hyper
		hypers, err = hyperAdmin.GetHypersByHostids(ctx, hostids)
		if err != nil {
			logger.Errorf("Failed to get hypers for migration %s: %v", migration.UUID, err)
			return
		}
		if hyper, ok := hypers[migration.SourceHyper]; ok {
			migrationResp.SourceHyperName = hyper.Hostname
		}
		if hyper, ok := hypers[migration.TargetHyper]; ok {
			migrationResp.TargetHyperName = hyper.Hostname
		}
	}

	// phases 组装
	migrationResp.Phases = make([]*MigrationPhaseResponse, len(migration.Phases))
	for i, phase := range migration.Phases {
		migrationResp.Phases[i] = &MigrationPhaseResponse{
			Name:      phase.Name,
			Summary:   phase.Summary,
			Status:    string(phase.Status),
			Message:   phase.Message,
			CreatedAt: phase.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: phase.UpdatedAt.Format(TimeStringForMat),
		}
	}

	return
}
