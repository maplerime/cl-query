/**
 * Licensed Materials - Property of PEG TECH INC
 *
 * (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
 *
 * Contributors:
 *    bryan@raksmart.com - Initial implementation
 *
 *
 * Purpose: CORS middleware unit test
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

func TestCors(t *testing.T) {
	// Create a gin engine with the Cors middleware
	r := gin.Default()
	r.Use(Cors())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Define test cases
	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "Allowed Origin 127.0.0.1",
			origin:         "http://127.0.0.1",
			expectedOrigin: "http://127.0.0.1",
		},
		{
			name:           "Allowed Origin localhost",
			origin:         "http://localhost",
			expectedOrigin: "http://localhost",
		},
		{
			name:           "Not Allowed Origin",
			origin:         "http://notallowed.com",
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)

			// Record the response
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			// Check the response headers for CORS
			assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
