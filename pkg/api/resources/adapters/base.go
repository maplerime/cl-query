/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Base adapter for resource queries
 *
**/

package adapters

import (
	"context"
	"fmt"
	"web/src/model"

	"github.com/gin-gonic/gin"
	"github.com/maplerime/cl-query/pkg/services"
	"github.com/maplerime/cl-query/utils/logging"

	"github.com/maplerime/cl-query/pkg/common"
)

var logger = logging.MustGetLogger("adapters")

var orgAdmin = &services.OrgAdmin{}
var hyperAdmin = &services.HyperAdmin{}

// ResourceQueryRequest 统一资源查询请求
type ResourceQueryRequest struct {
	ResourceType string                 `json:"resource_type" binding:"required" example:"instance"` // 资源类型
	Offset       int                    `json:"offset" example:"0"`                                  // 偏移量
	Limit        int                    `json:"limit" example:"16"`                                  // 限制数量
	Order        string                 `json:"order" example:"created_at"`                          // 排序字段
	Filters      map[string]interface{} `json:"filters,omitempty"`                                   // 查询过滤条件
}

// ResourceAdapter 资源适配器接口
type ResourceAdapter interface {
	List(c *gin.Context, req *ResourceQueryRequest) (interface{}, error)
	Get(c *gin.Context, id string) (interface{}, error)
	MakeQuery(c *gin.Context, filters map[string]interface{}) (string, error)
	ValidateRequest(req *ResourceQueryRequest) error
	CheckPermission(ctx context.Context) error
	NormalizeParams(req *ResourceQueryRequest)
}

// ParseFilters 泛型解析和验证过滤器
func ParseFilters[T any](c *gin.Context) (*T, error) {
	wrapper := struct {
		Filters T `json:"filters"`
	}{}

	if err := c.ShouldBindJSON(&wrapper); err != nil {
		return nil, err
	}

	return &wrapper.Filters, nil
}

// BaseAdapter 基础适配器，提供公共功能
type BaseAdapter struct{}

// ValidateRequest 验证请求参数
func (b *BaseAdapter) ValidateRequest(req *ResourceQueryRequest) error {
	if req.ResourceType == "" {
		return fmt.Errorf("resource_type is required")
	}
	if req.Limit < 0 {
		return fmt.Errorf("limit must be non-negative")
	}
	if req.Offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	return nil
}

// CheckPermission 检查权限（基础实现）
func (b *BaseAdapter) CheckPermission(ctx context.Context) error {
	memberShip := common.GetMemberShip(ctx)
	permit := memberShip.CheckPermission(model.Reader)
	if !permit {
		return fmt.Errorf("not authorized for this operation")
	}
	return nil
}

// NormalizeParams 标准化参数
func (b *BaseAdapter) NormalizeParams(req *ResourceQueryRequest) {
	if req.Limit == 0 {
		req.Limit = 16
	}
	if req.Order == "" {
		req.Order = "-created_at"
	}
}
