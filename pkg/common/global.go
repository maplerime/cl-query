/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: global variables
 *
**/

package common

import (
	"github.com/casbin/casbin/v2"
	"github.com/maplerime/cl-query/pkg/conf"
)

const (
	HH_RESOURCE_USER = "X-Resource-User"
	HH_RESOURCE_ORG  = "X-Resource-Org"

	CTX_USER         = "ptc_user"
	CTX_ORG          = "ptc_org"
	CTX_REQUEST_BODY = "ptc_request_body"
	TimeStringForMat = "2006-01-02 15:04:05"
	SVCID            = "SVCID"
	Version          = "Version"

	MiddlewareRequestIdCtxKey = "RequestId"
	MiddlewareTraceIdCtxKey   = "TraceId"
	MiddlewareSpanIdCtxKey    = "SpanId"

	TokenIssuer = "PetaCloud Query Service"

	VersionFile = "version"

	DefaultRateLimit = 60000
)

// ServiceConfig service config
type ServiceConfig struct {
	RPCSvc    conf.RPCSvcCfg
	Cache     conf.RedisCfg
	Runner    conf.RunnerConfig
	IAM       conf.IAM
	Database  conf.DBConfig
	Token     conf.Token
	DebugMode bool
	RateLimit int
}

func (c *ServiceConfig) GetAllowedOrigins() []string {
	if c != nil && c.RPCSvc.HTTPSvcCfg.AllowedOrigins != nil {
		return c.RPCSvc.HTTPSvcCfg.AllowedOrigins
	}
	return nil
}

var (
	CasbinEnforcer *casbin.Enforcer
	Config         *ServiceConfig
)
