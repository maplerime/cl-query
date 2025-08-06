/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: common functions
 *
**/

package common

import (
	"github.com/gin-gonic/gin"

	"github.com/maplerime/cl-query/utils/logging"
)

var logger = logging.MustGetLogger("common")

type BaseReference struct {
	ID   string `json:"id,omitempty" binding:"omitempty,uuid"`
	Name string `json:"name,omitempty" binding:"omitempty,min=2,max=32"`
}

type ResourceReference struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Owner     string         `json:"owner,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Region    *BaseReference `json:"region,omitempty"`
}

type BaseID struct {
	ID string `json:"id" binding:"required,uuid"`
}

func ErrorResponse(c *gin.Context, code int, errorMsg string, err error) {
	logger.Errorf("%s, %v\n", errorMsg, err)
	if err != nil {
		errorMsg = errorMsg + ": " + err.Error()
	}
	c.JSON(code, &APIError{ErrorMessage: errorMsg})
	return
}

type APIError struct {
	ErrorMessage string `json:"error_message"`
}
