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

	"github.com/gin-gonic/gin"
	gologging "github.com/op/go-logging"
	"github.com/stretchr/testify/assert"
)

type StringBackend struct {
	logs []string
}

func (b *StringBackend) Log(level gologging.Level, calldepth int, rec *gologging.Record) error {
	b.logs = append(b.logs, rec.Formatted(calldepth))
	return nil
}

func (b *StringBackend) GetLogs() []string {
	return b.logs
}

func TestLogger(t *testing.T) {
	// Initialize gin
	gin.SetMode(gin.TestMode)

	// Create a gin engine with the Logger middleware
	router := gin.New()
	router.Use(Logger())

	// Create a test route
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Create a test request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)

	// Mock the logger to capture log output
	logOutput := &StringBackend{}
	gologging.SetBackend(logOutput)

	// Serve the request
	router.ServeHTTP(w, req)

	// Check the response status code
	assert.Equal(t, http.StatusOK, w.Code)

	// Check the log output
	logs := logOutput.GetLogs()
	assert.NotEmpty(t, logs)
	assert.Contains(t, logs[0], "Request Method: GET")
	assert.Contains(t, logs[0], "Path: /test")
	assert.Contains(t, logs[0], "Status: 200")
	assert.Contains(t, logs[0], "Duration: ")
	assert.Contains(t, logs[0], "IP: ")
	assert.Contains(t, logs[0], "User-Agent: ")
	assert.Contains(t, logs[0], "Errors: ")
}
