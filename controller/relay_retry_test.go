package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestShouldRetryStopsWhenClientRequestDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	cancel()
	ctx.Request = req.WithContext(reqCtx)

	upstreamErr := types.NewErrorWithStatusCode(
		fmt.Errorf("upstream error"),
		types.ErrorCodeBadResponse,
		http.StatusInternalServerError,
	)

	assert.False(t, shouldRetry(ctx, upstreamErr, 1))
}

func TestShouldRetryTaskRelayStopsWhenClientRequestDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	reqCtx, cancel := context.WithCancel(req.Context())
	cancel()
	ctx.Request = req.WithContext(reqCtx)

	taskErr := &dto.TaskError{
		Code:       "upstream_error",
		Message:    "upstream error",
		StatusCode: http.StatusInternalServerError,
	}

	assert.False(t, shouldRetryTaskRelay(ctx, 1, taskErr, 1))
}
