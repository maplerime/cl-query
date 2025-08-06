/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: CORS middleware
 *
**/
package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/maplerime/cl-query/pkg/common"
)

// Cors is a middleware that handles CORS requests
// It allows all origins, methods, headers, and credentials
// It also sets the max age to 12 hours
// The allowed origins should be configured in the configuration file

var (
	DefaultAllowedOrigins = []string{
		"http://127.0.0.1",
		"http://127.0.0.1:8001",
		"http://localhost",
		"http://localhost:8001",
		"http://0.0.0.0:8088",
	}
)

func Cors() gin.HandlerFunc {
	//load allowed origins from configuration file
	var allowedOrigins []string
	if common.Config != nil {
		allowedOrigins = common.Config.GetAllowedOrigins()
	}
	if len(allowedOrigins) == 0 {
		allowedOrigins = DefaultAllowedOrigins
	}
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
