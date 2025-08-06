/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Rate limiter middleware
 *
**/
package middleware

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/maplerime/cl-query/utils/logging"

	"github.com/gin-gonic/gin"
)

var logger = logging.MustGetLogger("api/middleware")

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// End timer
		duration := time.Since(start)
		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		message := fmt.Sprintf("Request Method: %s, Path: %s, Status: %d, Content-Length: %s, Content-Type: %s, Duration: %s, IP: %s, Origin: %s, User-Agent: %s, Errors: %s, Body:\n%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			c.Request.Header.Get("Content-Length"),
			c.ContentType(),
			duration,
			c.ClientIP(),
			c.Request.Header.Get("Origin"),
			c.Request.UserAgent(),
			c.Errors.ByType(gin.ErrorTypePrivate).String(),
			string(body),
		)
		if c.Writer.Status() >= 400 {
			logger.Error(message)
		} else {
			logger.Info(message)
		}
	}
}
