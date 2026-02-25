/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Volume resource adapter
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

type VolumeResponse struct {
	*ResourceReference
	Path      string         `json:"path"`
	Size      int32          `json:"size"`
	Format    string         `json:"format"`
	Status    string         `json:"status"`
	Target    string         `json:"target"`
	Href      string         `json:"href"`
	Booting   bool           `json:"booting"`
	Instance  *BaseReference `json:"instance"`
	IopsLimit int32          `json:"iops_limit"`
	IopsBurst int32          `json:"iops_burst"`
	BpsLimit  int32          `json:"bps_limit"`
	BpsBurst  int32          `json:"bps_burst"`
	PoolName  string         `json:"pool_name"`
}

type VolumeListResponse struct {
	Offset  int               `json:"offset"`
	Total   int               `json:"total"`
	Limit   int               `json:"limit"`
	Volumes []*VolumeResponse `json:"volumes"`
}

type VolumeInfoResponse struct {
	*ResourceReference
	Target  string `json:"target"`
	Booting bool   `json:"booting"`
}

type VolumeFilters struct {
	InstanceID string   `json:"instance_id,omitempty" binding:"omitempty,uuid"`
	Name       string   `json:"name,omitempty"`
	Status     string   `json:"status,omitempty" binding:"omitempty,oneof=available in-use error"`
	UUIDs      []string `json:"uuids,omitempty" binding:"omitempty,dive,uuid"`
	VolumeType string   `json:"volume_type,omitempty" binding:"omitempty,oneof=data boot all"` // 卷类型: data, boot, all
}

type VolumeAdapter struct {
	BaseAdapter
	service           *services.VolumeAdmin
	instanceService   *services.InstanceAdmin
	dictionaryService *services.DictionaryAdmin
}

func NewVolumeAdapter() *VolumeAdapter {
	logger.Debug("Creating new Volume adapter")
	return &VolumeAdapter{
		service:           &services.VolumeAdmin{},
		dictionaryService: &services.DictionaryAdmin{},
	}
}

func (a *VolumeAdapter) MakeQuery(c *gin.Context, filtersMap map[string]interface{}) (string, error) {
	logger.Debugf("Processing Volume filters: %+v", filtersMap)

	filters, err := ParseFilters[VolumeFilters](c)
	if err != nil {
		return "", err
	}

	var conditions []string

	// 名称查询
	if filters.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name like '%%%s%%'", filters.Name))
		logger.Debugf("Added name filter: %s", filters.Name)
	}

	// 状态查询
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = '%s'", filters.Status))
		logger.Debugf("Added status filter: %s", filters.Status)
	}

	// 实例查询
	if filters.InstanceID != "" {
		inst, err := a.instanceService.GetInstanceByUUID(c.Request.Context(), filters.InstanceID)
		if err != nil {
			return "", err
		}
		conditions = append(conditions, fmt.Sprintf("instance_id = %d", inst.ID))
		logger.Debugf("Added instance_id filter: %s -> instance_id = %d", filters.InstanceID, inst.ID)
	}

	// UUIDs 查询
	if filters.UUIDs != nil {
		if len(filters.UUIDs) > 0 {
			conditions = append(conditions, fmt.Sprintf("uuid IN ('%s')", strings.Join(filters.UUIDs, "','")))
			logger.Debugf("Added UUIDs filter: %v", filters.UUIDs)
		} else {
			conditions = append(conditions, "1=0")
		}
	}

	// 卷类型查询
	if filters.VolumeType == "" {
		filters.VolumeType = "data"
	}
	booting := ""
	if filters.VolumeType == "data" {
		booting = fmt.Sprintf("booting=%t", false)
	} else if filters.VolumeType == "boot" {
		booting = fmt.Sprintf("booting=%t", true)
	} else if filters.VolumeType == "all" {
		booting = ""
	} else {
		return "", NewCLError(ErrInvalidParameter, fmt.Sprintf("Invalid volume type %s", filters.VolumeType), nil)
	}
	if booting != "" {
		conditions = append(conditions, booting)
		logger.Debugf("Added volume_type filter: %s", booting)
	}

	if len(conditions) > 0 {
		query := strings.Join(conditions, " AND ")
		logger.Debugf("Generated query: %s", query)
		return query, nil
	}

	logger.Debug("No valid conditions found, returning empty query")
	return "", nil
}

func (a *VolumeAdapter) List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error) {
	logger.Debugf("Starting Volume list query with request: %+v", req)

	ctx := c.Request.Context()

	// 处理过滤条件
	query, err := a.MakeQuery(c, req.Filters)
	if err != nil {
		logger.Errorf("Failed to process filters: %v", err)
		return nil, err
	}

	// 调用 service 层
	logger.Debugf("Calling service layer with query: %s", query)
	total, volumes, err := a.service.ListVolume(ctx, int64(req.Offset), int64(req.Limit), req.Order, query)
	if err != nil {
		logger.Errorf("Service layer query failed: %v", err)
		return nil, err
	}

	logger.Infof("Successfully retrieved %d Volumes (total: %d)", len(volumes), total)

	volumeListResp := &VolumeListResponse{
		Offset: req.Offset,
		Total:  int(total),
		Limit:  len(volumes),
	}

	poolNameMap := a.getPoolNameMap(ctx)

	volumeList := make([]*VolumeResponse, volumeListResp.Limit)
	for i, volume := range volumes {
		volumeResp, err := a.getVolumeResponse(ctx, volume, poolNameMap)
		if err != nil {
			return nil, err
		}
		volumeList[i] = volumeResp
	}

	volumeListResp.Volumes = volumeList
	logger.Debugf("List volumes successfully, %+v", volumeListResp)
	return volumeListResp, nil
}

func (a *VolumeAdapter) Get(c *gin.Context, id string) (interface{}, error) {
	logger.Debugf("Starting Volume get query with ID: %s", id)

	ctx := c.Request.Context()
	volume, err := a.service.GetVolumeByUUID(ctx, id)
	if err != nil {
		return nil, err
	}

	poolNameMap := a.getPoolNameMap(ctx)

	result, err := a.getVolumeResponse(ctx, volume, poolNameMap)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Get volume successfully: %+v", result)
	return result, nil
}

func (a *VolumeAdapter) getPoolNameMap(ctx context.Context) map[string]string {
	poolNameMap := map[string]string{}
	_, dictionaries, err := a.dictionaryService.List(ctx, 0, 10, "", "category='storage_pool'")
	if err != nil {
		logger.Warningf("Failed to fetch storage pool names: %v", err)
		return poolNameMap
	}
	for _, d := range dictionaries {
		poolNameMap[d.Value] = d.Name
	}
	return poolNameMap
}

func (a *VolumeAdapter) getVolumeResponse(ctx context.Context, volume *model.Volume, poolNameMap map[string]string) (volumeResp *VolumeResponse, err error) {
	owner := orgAdmin.GetOrgName(ctx, volume.Owner)
	poolName := ""
	if volume.PoolID != "" {
		poolName = poolNameMap[volume.PoolID]
	}
	volumeResp = &VolumeResponse{
		ResourceReference: &ResourceReference{
			ID:        volume.UUID,
			Name:      volume.Name,
			Owner:     owner,
			CreatedAt: volume.CreatedAt.Format(TimeStringForMat),
			UpdatedAt: volume.UpdatedAt.Format(TimeStringForMat),
		},
		Path:      volume.Path,
		Size:      volume.Size,
		Status:    string(volume.Status),
		Target:    volume.Target,
		Href:      volume.Href,
		IopsLimit: volume.IopsLimit,
		IopsBurst: volume.IopsBurst,
		BpsLimit:  volume.BpsLimit,
		BpsBurst:  volume.BpsBurst,
		Booting:   volume.Booting,
		PoolName:  poolName,
	}
	if volume.Instance == nil {
		volumeResp.Instance = nil
	} else {
		volumeResp.Instance = &BaseReference{
			ID:   volume.Instance.UUID,
			Name: volume.Instance.Hostname,
		}
	}
	return
}
