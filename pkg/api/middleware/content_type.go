/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    jack@raksmart.com - Check content type middleware
 *
 *
 * Purpose: QueryService Check content type middleware
 *
**/

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ContentType() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Ignore GET and DELETE requests
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodDelete {
			c.Next()
			return
		}

		// Check if the Content-Type is application/json
		if c.ContentType() != "application/json" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Content-Type must be application/json",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
