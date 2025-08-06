/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Query Service metadata handler
 *
**/

package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/maplerime/cl-query/pkg/common"
)

func GetMetadata(c *gin.Context) {
	metadata := common.GetSvcMetadata()
	c.JSON(http.StatusOK, metadata)
}
