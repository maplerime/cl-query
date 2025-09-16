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
	"errors"
	"fmt"
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
		var clErr *CLError
		if errors.As(err, &clErr) {
			c.JSON(code, &APIError{
				ErrorCode:    int(clErr.Code),
				ErrorCodeStr: clErr.Code.String(),
				ErrorMessage: clErr.Message,
			})
			return
		}
		errorMsg = errorMsg + ": " + err.Error()
	}
	c.JSON(code, &APIError{
		ErrorCode:    code,
		ErrorMessage: errorMsg,
	})
}

type APIError struct {
	ErrorCode    int    `json:"error_code"`
	ErrorCodeStr string `json:"error_code_str,omitempty"`
	ErrorMessage string `json:"error_message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("CLError: code=%d, message=%s", e.ErrorCode, e.ErrorMessage)
}
