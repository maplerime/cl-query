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
	"strconv"
)

type StatisticsAPI struct{}

func NewStatisticsAPI() *StatisticsAPI {
	return &StatisticsAPI{}
}

type StoragePoolData struct {
	PoolName    string `json:"pool_name"`     // 存储池名称
	PhySize     uint64 `json:"phy_size"`      // 数据量
	UsedSize    uint64 `json:"used_size"`     // 已使用
	OverRate    uint64 `json:"over_rate"`     // 超分比
	OverPhySize uint64 `json:"over_phy_size"` // 超分后裸容量
	OverUsed    uint64 `json:"over_used"`     // 超分后已使用
}

type StorageData struct {
	NodeCount   int64              `json:"node_count"`   // 存储节点数
	VolumeCount int64              `json:"volume_count"` // 存储卷数
	PhySize     uint64             `json:"phy_size"`     // 裸容量
	UsedSize    uint64             `json:"used_size"`    // 已使用
	UnusedSize  uint64             `json:"unused_size"`  // 未使用
	DataSize    uint64             `json:"data_size"`    // 数据量
	PoolData    []*StoragePoolData `json:"pool_data"`    // 存储池信息
}

type ZoneData struct {
	*BaseReference
	Cpu      int64 `json:"cpu"`
	CpuTotal int64 `json:"cpu_total"`
}

type ResourceStatisticsResponse struct {
	HyperCount       int64            `json:"hyper_count"`
	ZoneCount        int64            `json:"zone_count"`
	InstanceCount    int64            `json:"instance_count"`
	InstanceByStatus map[string]int64 `json:"instance_by_status"`
	StorageData      *StorageData     `json:"storage_data"`
	ZoneData         []*ZoneData      `json:"zone_data"`
}

// Resources
// @Summary 获取资源统计
// @Description 获取资源统计
// @Tags Statistics
// @Produce json
// @Failure 400 {object} APIError "请求参数错误"
// @Failure 404 {object} APIError "实例不存在"
// @Failure 500 {object} APIError "服务器内部错误"
// @Router /statistics/resources [get]
func (api *StatisticsAPI) Resources(c *gin.Context) {
	ctx := c.Request.Context()

	// 计算节点总数
	hyperAdmin := &services.HyperAdmin{}
	hyperCount, err := hyperAdmin.Count(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get instance count", err)
		return
	}

	// zone分组
	zoneAdmin := &services.ZoneAdmin{}
	zoneCount, zones, err := zoneAdmin.List(ctx, 0, 100, "", "")
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get zones", err)
		return
	}
	zoneIDs := make([]int64, len(zones))
	for i, zone := range zones {
		zoneIDs[i] = zone.ID
	}

	// hyper列表
	hyperByZone, err := hyperAdmin.GetHypersByZoneIDs(ctx, zoneIDs)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get hypers by zone IDs", err)
		return
	}

	// zone统计
	zoneData := make([]*ZoneData, len(zones))
	for i, zone := range zones {
		zoneData[i] = &ZoneData{
			BaseReference: &BaseReference{
				ID:   strconv.FormatInt(zone.ID, 10),
				Name: zone.Name,
			},
		}
		if hypers, exists := hyperByZone[zone.ID]; exists {
			for _, hyper := range hypers {
				if hyper.Resource != nil {
					zoneData[i].Cpu += hyper.Resource.Cpu
					zoneData[i].CpuTotal += hyper.Resource.CpuTotal
				}
			}
		}
	}

	// instance各状态统计
	instanceAdmin := &services.InstanceAdmin{}
	instanceCount, instanceByStatus, err := instanceAdmin.GetStatusStatistics(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get instance statistics", err)
		return
	}

	// WDS
	storageData := &StorageData{}
	wdsAdmin := &services.WdsAdmin{}
	wdsServers, err := wdsAdmin.GetServers("STORE")
	if err != nil {
		logger.Errorf("Get WDS storage nodes failed: %+v", err)
	} else {
		storageData.NodeCount = wdsServers.Data.TotalCount
	}

	// 卷总数
	volumeCount, err := wdsAdmin.GetVolumeCount()
	if err != nil {
		logger.Errorf("Get WDS volume count failed: %+v", err)
	} else {
		storageData.VolumeCount = volumeCount
	}

	// 存储池
	pools, err := wdsAdmin.GetPools()
	if err != nil {
		logger.Errorf("Get WDS pools failed: %+v", err)
		storageData.PoolData = make([]*StoragePoolData, 0)
	} else {
		// 裸容量
		phySize := uint64(0)
		// 数据量（有效容量）
		dataSize := uint64(0)
		// 已使用
		usedSize := uint64(0)
		poolData := make([]*StoragePoolData, pools.Data.TotalCount)
		for i, pool := range pools.Data.List {
			phySize += pool.PhySize * pool.ReplicateSize
			usedSize += pool.PhyUsedSize * pool.ReplicateSize
			dataSize += pool.PhySize
			poolData[i] = &StoragePoolData{
				PoolName:    pool.ClusterName,
				PhySize:     pool.PhySize * pool.ReplicateSize,
				UsedSize:    pool.PhyUsedSize * pool.ReplicateSize,
				OverRate:    pool.ThinProvisioning,
				OverUsed:    pool.VolumeSizeSum * pool.ReplicateSize,
				OverPhySize: pool.PhySize * pool.ReplicateSize * pool.ThinProvisioning,
			}
		}
		storageData.PhySize = phySize
		storageData.DataSize = dataSize
		storageData.UsedSize = usedSize
		storageData.UnusedSize = phySize - usedSize
		storageData.PoolData = poolData
	}

	c.JSON(http.StatusOK, ResourceStatisticsResponse{
		HyperCount:       hyperCount,
		ZoneCount:        zoneCount,
		InstanceCount:    instanceCount,
		InstanceByStatus: instanceByStatus,
		StorageData:      storageData,
		ZoneData:         zoneData,
	})

}
