package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func ListChannelModelDescriptions(c *gin.Context) {
	var rows []model.ChannelModelDescription
	query := model.DB.Order("channel_id ASC, model_name ASC")
	if channelID := c.Query("channel_id"); channelID != "" {
		id, err := strconv.Atoi(channelID)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		query = query.Where("channel_id = ?", id)
	}
	if err := query.Find(&rows).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

func UpsertChannelModelDescription(c *gin.Context) {
	var row model.ChannelModelDescription
	if err := c.ShouldBindJSON(&row); err != nil {
		common.ApiError(c, err)
		return
	}
	row.ModelName = strings.TrimSpace(row.ModelName)
	row.Description = strings.TrimSpace(row.Description)
	if row.ChannelID <= 0 || row.ModelName == "" {
		common.ApiErrorMsg(c, "channel_id and model_name are required")
		return
	}
	var channel model.Channel
	if err := model.DB.First(&channel, row.ChannelID).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpsertChannelModelDescription(&row); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidatePricingCache()
	common.ApiSuccess(c, &row)
}

func DeleteChannelModelDescription(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Delete(&model.ChannelModelDescription{}, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidatePricingCache()
	c.JSON(http.StatusOK, gin.H{"success": true})
}
