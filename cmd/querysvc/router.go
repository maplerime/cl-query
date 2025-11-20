/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: query service api router
 *
**/

package main

import (
	"context"
	"github.com/maplerime/cl-query/pkg/api"

	"github.com/gin-gonic/gin"
	"github.com/maplerime/cl-query/pkg/api/resources"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"golang.org/x/time/rate"

	"github.com/maplerime/cl-query/docs"
	"github.com/maplerime/cl-query/pkg/api/middleware"
	"github.com/maplerime/cl-query/pkg/common"
)

func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// setup context in gin

		// ...
		c.Next()
	}
}

// @title           PETACLOUID Query Service API
// @version         1.0.0
// @description     PetaCloud Query Service
// @termsOfService  https://raksmart.com/dev/peta-cloud/terms/

// @contact.name   API Support
// @contact.url    https://raksmart.com/dev/support
// @contact.email  dev-support@raksmart.com

// @BasePath  /api/v1

// @securityDefinitions.apikey JwtAuth
// @in header
// @name Authorization

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @externalDocs.description  PetaCloud API documentation
// @externalDocs.url          https://raksmart.com/dev/peta-cloud/docs
func NewRouter(ctx *context.Context) *gin.Engine {
	logger.Debugf("Entry: context: %+v", ctx)
	r := gin.Default()
	r.Use(ContextMiddleware())
	r.Use(middleware.Logger())
	r.Use(middleware.Cors())
	rateLimit := common.Config.RateLimit
	if rateLimit <= 0 {
		rateLimit = common.DefaultRateLimit
	}
	// convert to second limit
	limit := rate.Limit(float64(rateLimit) / 60.0)
	r.Use(middleware.RateLimiter(limit, rateLimit))
	r.Use(middleware.ContentType())
	r.GET("/api/v1/query", api.GetMetadata)

	// 创建IP树API实例
	ipTreeAPI := api.NewIPTreeAPI()
	// 创建用量API实例
	usageAPI := api.NewUsageAPI()
	statisticsAPI := api.NewStatisticsAPI()

	v1 := r.Group("/api/v1/query").Use(middleware.Authorize())
	{
		v1.POST("/resources", resources.QueryResources)
		v1.GET("/resources/:resource_type/:id", resources.GetResource)
		v1.GET("/instances/:id/subnet-ip-tree", ipTreeAPI.GetInstanceIPTree)
		v1.GET("/statistics/resources", statisticsAPI.Resources)
		v1.GET("/usage/summary", usageAPI.Summary)
	}

	docs.SwaggerInfo.BasePath = "/api/v1"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler, ginSwagger.DocExpansion("none")))

	return r
}
