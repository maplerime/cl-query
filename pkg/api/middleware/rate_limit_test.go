/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: Rate limiter middleware unit test
 *
**/
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {
	// Create a gin engine with the RateLimiter middleware
	r := gin.Default()
	r.Use(RateLimiter(rate.Every(1*time.Second), 1)) // 1 request per second
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Define test cases
	tests := []struct {
		name       string
		delay      time.Duration
		statusCode int
		body       string
	}{
		{
			name:       "First Request",
			delay:      0,
			statusCode: http.StatusOK,
			body:       `{"message":"success"}`,
		},
		{
			name:       "Second Request Exceeding Limit",
			delay:      0,
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":"Too Many Requests"}`,
		},
		{
			name:       "Third Request After Delay",
			delay:      1 * time.Second,
			statusCode: http.StatusOK,
			body:       `{"message":"success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wait for the specified delay before the next request
			time.Sleep(tt.delay)

			// Create a test request
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)

			// Record the response
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Check the response status code
			assert.Equal(t, tt.statusCode, w.Code)
		})
	}
}
