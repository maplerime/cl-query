/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: API Key authentication middleware unit test
 *
**/
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestContentType(t *testing.T) {

	// Create a gin engine with the ContentType middleware
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.Use(ContentType())
	r.Any("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	r.Handler()

	// Define test cases
	tests := []struct {
		name        string
		method      string
		contentType string
		statusCode  int
		body        string
	}{
		{
			name:        "Valid Content Type",
			method:      http.MethodGet,
			contentType: "application/json",
			statusCode:  http.StatusOK,
			body:        `{"message":"success"}`,
		},
		{
			name:        "Invalid Content Type",
			method:      http.MethodPost,
			contentType: "application/xml",
			statusCode:  http.StatusBadRequest,
			body:        `{"error":"Content-Type must be application/json"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Content-Type", tt.contentType)

			// Record the response
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Check the response status code and body
			assert.Equal(t, tt.statusCode, w.Code)
			assert.JSONEq(t, tt.body, w.Body.String())
		})
	}
}
