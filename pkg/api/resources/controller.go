/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Unified resource query controller
 *
**/

package resources

import (
	"bytes"
	"io"
	"net/http"

	. "github.com/maplerime/cl-query/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/maplerime/cl-query/pkg/api/resources/adapters"
	"github.com/maplerime/cl-query/utils/logging"
)

var logger = logging.MustGetLogger("resource-controller")

// QueryResources 统一资源查询接口
// @Summary 统一资源查询
// @Description 通过资源类型查询不同类型的资源列表
// @Tags 资源查询
// @Accept json
// @Produce json
// @Param request body adapters.ResourceQueryRequest true "查询请求"
// @Success 200 {object} interface{} "查询结果"
// @Failure 400 {object} APIError "请求参数错误"
// @Failure 500 {object} APIError "服务器内部错误"
// @Router /query/v1/resources [post]
func QueryResources(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	var req adapters.ResourceQueryRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}
	logger.Debugf("Resource query request: %+v", req)

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 创建Adapter
	adapter, err := DefaultFactory.CreateAdapter(req.ResourceType)
	if err != nil {
		logger.Errorf("Unsupported resource type '%s'", req.ResourceType)
		ErrorResponse(c, http.StatusBadRequest, "Unsupported resource type", nil)
		return
	}

	// 统一验证请求参数
	if err = adapter.ValidateRequest(&req); err != nil {
		logger.Errorf("Request validation failed: %v", err)
		ErrorResponse(c, http.StatusBadRequest, "Invalid request parameters", err)
		return
	}

	// 统一检查权限
	ctx := c.Request.Context()
	if err = adapter.CheckPermission(ctx); err != nil {
		logger.Errorf("Permission check failed: %v", err)
		ErrorResponse(c, http.StatusForbidden, "Access denied", err)
		return
	}

	adapter.NormalizeParams(&req)
	logger.Debugf("Normalized parameters: offset=%d, limit=%d, order=%s", req.Offset, req.Limit, req.Order)

	// 执行查询
	result, err := adapter.List(c, &req)
	if err != nil {
		ErrorResponse(c, http.StatusInternalServerError, "Failed to query resources", err)
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, result)
}

// GetResource 获取单个资源详情
// @Summary 获取单个资源详情
// @Description 通过资源类型和 ID 获取单个资源的详细信息
// @Tags 资源查询
// @Produce json
// @Param resource_type path string true "资源类型" Enums(instance,secgroup,secrule,subnet,volume,floatingip)
// @Param id path string true "资源ID（UUID）"
// @Success 200 {object} interface{} "资源详情"
// @Failure 400 {object} APIError "请求参数错误"
// @Failure 404 {object} APIError "资源不存在"
// @Failure 500 {object} APIError "服务器内部错误"
// @Router /query/v1/resources/{resource_type}/{id} [get]
func GetResource(c *gin.Context) {
	resourceType := c.Param("resource_type")
	id := c.Param("id")

	logger.Debugf("Resource get request: type=%s, id=%s", resourceType, id)

	// 验证参数
	if resourceType == "" {
		ErrorResponse(c, http.StatusBadRequest, "Resource type is required", nil)
		return
	}
	if id == "" {
		ErrorResponse(c, http.StatusBadRequest, "Resource ID is required", nil)
		return
	}

	// 创建 Adapter
	adapter, err := DefaultFactory.CreateAdapter(resourceType)
	if err != nil {
		logger.Errorf("Unsupported resource type '%s'", resourceType)
		ErrorResponse(c, http.StatusBadRequest, "Unsupported resource type", nil)
		return
	}

	// 统一检查权限
	ctx := c.Request.Context()
	if err = adapter.CheckPermission(ctx); err != nil {
		logger.Errorf("Permission check failed: %v", err)
		ErrorResponse(c, http.StatusForbidden, "Access denied", err)
		return
	}

	// 执行查询
	result, err := adapter.Get(c, id)
	if err != nil {
		logger.Errorf("Failed to get resource: %v", err)
		ErrorResponse(c, http.StatusInternalServerError, "Failed to get resource", err)
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, result)
}
