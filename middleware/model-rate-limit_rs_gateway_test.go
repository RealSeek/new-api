package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestModelRequestRateLimitSkipsRSGateway(t *testing.T) {
	originalEnabled := setting.ModelRequestRateLimitEnabled
	setting.ModelRequestRateLimitEnabled = true
	t.Cleanup(func() {
		setting.ModelRequestRateLimitEnabled = originalEnabled
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeRSGateway)
	})
	router.Use(ModelRequestRateLimit())
	router.GET("/v1/models", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/models", nil))

	assert.Equal(t, http.StatusNoContent, response.Code)
}
