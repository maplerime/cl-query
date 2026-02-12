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
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/maplerime/cl-query/pkg/common"
	"github.com/maplerime/cl-query/pkg/services"
)

type StatisticsAPI struct{}

func NewStatisticsAPI() *StatisticsAPI {
	return &StatisticsAPI{}
}

type StoragePoolData struct {
	MediaType   string `json:"media_type"`    // 存储介质
	PoolName    string `json:"pool_name"`     // 存储池名称
	PhySize     uint64 `json:"phy_size"`      // 数据量
	UsedSize    uint64 `json:"used_size"`     // 已使用
	OverRate    uint64 `json:"over_rate"`     // 超分比
	OverPhySize uint64 `json:"over_phy_size"` // 超分后裸容量
	OverUsed    uint64 `json:"over_used"`     // 超分后已使用
}

type StorageData struct {
	NodeCount     int64              `json:"node_count"`      // 存储节点数
	HDDNodeCount  int64              `json:"hdd_node_count"`  // HDD存储节点数
	NVMENodeCount int64              `json:"nvme_node_count"` // NVME存储节点数
	VolumeCount   int64              `json:"volume_count"`    // 存储卷数
	PhySize       uint64             `json:"phy_size"`        // 裸容量
	UsedSize      uint64             `json:"used_size"`       // 已使用
	UnusedSize    uint64             `json:"unused_size"`     // 未使用
	DataSize      uint64             `json:"data_size"`       // 数据量
	PoolData      []*StoragePoolData `json:"pool_data"`       // 存储池信息
}

type ZoneData struct {
	*BaseReference
	HyperCount       int64            `json:"hyper_count"`        // 计算节点总数
	FreeHyperCount   float64          `json:"free_hyper_count"`   // 空闲计算节点数
	UsedHyperCount   float64          `json:"used_hyper_count"`   // 已用计算节点数
	CpuTotal         int64            `json:"cpu_total"`          // CPU总量
	CpuUsed          int64            `json:"cpu_used"`           // CPU使用
	CpuFree          int64            `json:"cpu_free"`           // CPU空闲
	InstanceCount    int64            `json:"instance_count"`     // 实例总数
	InstanceByStatus map[string]int64 `json:"instance_by_status"` // 实例数量按状态分组
}

type ResourceStatisticsResponse struct {
	ZoneCount        int64            `json:"zone_count"`         // 计算节点分组数
	HyperCount       int64            `json:"hyper_count"`        // 计算节点总数
	FreeHyperCount   float64          `json:"free_hyper_count"`   // 空闲计算节点数
	UsedHyperCount   float64          `json:"used_hyper_count"`   // 已用计算节点数
	CpuTotal         int64            `json:"cpu_total"`          // CPU总量
	CpuUsed          int64            `json:"cpu_used"`           // CPU使用
	CpuFree          int64            `json:"cpu_free"`           // CPU空闲
	InstanceCount    int64            `json:"instance_count"`     // 实例总数
	InstanceByStatus map[string]int64 `json:"instance_by_status"` // 实例数量按状态分组
	StorageData      *StorageData     `json:"storage_data"`       // 存储数据
	ZoneData         []*ZoneData      `json:"zone_data"`          // 分组数据
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

	// 实例统计
	instanceAdmin := &services.InstanceAdmin{}
	hyperStatusCounts, err := instanceAdmin.GetHyperStatusCounts(ctx)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get instance statistics", err)
		return
	}
	countsMap := make(map[int32]map[string]int64)
	for _, hyperStatus := range hyperStatusCounts {
		if _, exists := countsMap[hyperStatus.Hyper]; !exists {
			countsMap[hyperStatus.Hyper] = make(map[string]int64)
		}
		countsMap[hyperStatus.Hyper][hyperStatus.Status] = hyperStatus.Count
	}

	// zone统计
	cpuTotal := int64(0)
	cpuUsed := int64(0)
	cpuFree := int64(0)
	instanceCount := int64(0)
	instanceByStatus := map[string]int64{
		"pending":      0,
		"running":      0,
		"shut_off":     0,
		"paused":       0,
		"migrating":    0,
		"reinstalling": 0,
		"resizing":     0,
		"deleting":     0,
	}
	zoneData := make([]*ZoneData, len(zones))
	usedHyperCount := float64(0)
	freeHyperCount := float64(0)
	for i, zone := range zones {
		cpuCore := float64(88)
		if zone.Name == "zone2" {
			cpuCore = float64(256)
		}
		zoneData[i] = &ZoneData{
			BaseReference: &BaseReference{
				ID:   strconv.FormatInt(zone.ID, 10),
				Name: zone.Name,
			},
			InstanceByStatus: map[string]int64{
				"pending":      0,
				"running":      0,
				"shut_off":     0,
				"paused":       0,
				"migrating":    0,
				"reinstalling": 0,
				"resizing":     0,
				"deleting":     0,
			},
		}
		if hypers, exists := hyperByZone[zone.ID]; exists {
			for _, hyper := range hypers {
				zoneData[i].HyperCount++
				if hyper.Resource != nil {
					// zone下的CPU数据汇总
					zoneData[i].CpuFree += hyper.Resource.Cpu
					zoneData[i].CpuTotal += hyper.Resource.CpuTotal
					// 全部CPU数据汇总
					cpuFree += hyper.Resource.Cpu
					cpuTotal += hyper.Resource.CpuTotal
					// 挨个hyper统计已用和空闲计算节点,并追加到zone中
					usedHyper := float64(hyper.Resource.CpuTotal-hyper.Resource.Cpu) / float64(hyper.CpuOverRate) / cpuCore
					freeHyper := float64(hyper.Resource.Cpu) / float64(hyper.CpuOverRate) / cpuCore
					zoneData[i].UsedHyperCount += usedHyper
					zoneData[i].FreeHyperCount += freeHyper
				}
				// 统计实例数量
				if counts, exist := countsMap[hyper.Hostid]; exist {
					for status, count := range counts {
						zoneData[i].InstanceCount += count
						zoneData[i].InstanceByStatus[status] += count
						instanceCount += count
						instanceByStatus[status] += count
					}
				}
			}
		}
		zoneData[i].CpuUsed = zoneData[i].CpuTotal - zoneData[i].CpuFree
		cpuUsed += zoneData[i].CpuUsed
		usedHyperCount += zoneData[i].UsedHyperCount
		freeHyperCount += zoneData[i].FreeHyperCount
	}

	// WDS
	storageData := &StorageData{}
	wdsAdmin := &services.WdsAdmin{}
	wdsServers, err := wdsAdmin.GetServers("STORE")
	if err != nil {
		logger.Errorf("Get WDS storage nodes failed: %+v", err)
	} else {
		for _, server := range wdsServers.Data.List {
			hostname := strings.ToLower(server.HostName)
			if strings.Contains(hostname, "nvme") {
				// NVME存储节点数
				storageData.NVMENodeCount++
				storageData.NodeCount++
			} else if strings.Contains(hostname, "hdd") {
				// HDD存储节点数
				storageData.HDDNodeCount++
				storageData.NodeCount++
			}
		}

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
			usedSize += pool.PhyUsedSize
			dataSize += pool.PhySize
			poolData[i] = &StoragePoolData{
				MediaType:   pool.StorageMediaType,
				PoolName:    pool.ClusterName,
				PhySize:     pool.PhySize,
				UsedSize:    pool.PhyUsedSize,
				OverRate:    pool.ThinProvisioning,
				OverUsed:    pool.VolumeSizeSum,
				OverPhySize: pool.PhySize * pool.ThinProvisioning,
			}
		}
		storageData.PhySize = phySize
		storageData.DataSize = dataSize
		storageData.UsedSize = usedSize
		storageData.UnusedSize = dataSize - usedSize
		storageData.PoolData = poolData
	}

	c.JSON(http.StatusOK, ResourceStatisticsResponse{
		HyperCount:       hyperCount,
		FreeHyperCount:   freeHyperCount,
		UsedHyperCount:   usedHyperCount,
		ZoneCount:        zoneCount,
		CpuTotal:         cpuTotal,
		CpuUsed:          cpuUsed,
		CpuFree:          cpuFree,
		InstanceCount:    instanceCount,
		InstanceByStatus: instanceByStatus,
		StorageData:      storageData,
		ZoneData:         zoneData,
	})

}
