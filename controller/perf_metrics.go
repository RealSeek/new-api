package controller

import (
	"cmp"
	"net/http"
	"slices"
	"strconv"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

type perfGroupStatusItem struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Ratio         float64                  `json:"ratio"`
	Models        []string                 `json:"models"`
	EndpointTypes []constant.EndpointType  `json:"endpoint_types"`
	Status        *perfmetrics.GroupStatus `json:"status"`
}

// GetPerfMetricsGroups serves the public group status board: every group the
// visitor can select with its ratio, models and the last 24 hours of relay
// health. Groups without samples in the window keep a null status.
func GetPerfMetricsGroups(c *gin.Context) {
	userGroup := ""
	if userId, exists := c.Get("id"); exists {
		if user, err := model.GetUserCache(userId.(int)); err == nil {
			userGroup = user.Group
		}
	}
	groupRatios := ratio_setting.GetGroupRatioCopy()
	usableGroups := service.GetUserUsableGroups(userGroup)
	names := make([]string, 0, len(usableGroups))
	for name := range usableGroups {
		// Same rule as token group selection: auto is not a channel group and a
		// group without a ratio cannot be selected.
		if _, ok := groupRatios[name]; ok && name != "auto" {
			names = append(names, name)
		}
	}

	result, err := perfmetrics.QueryGroupStatus(names)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	groups := make([]perfGroupStatusItem, 0, len(names))
	for _, name := range names {
		ratio := groupRatios[name]
		if userRatio, ok := ratio_setting.GetGroupGroupRatio(userGroup, name); ok {
			ratio = userRatio
		}
		item := perfGroupStatusItem{
			Name:          name,
			Description:   usableGroups[name],
			Ratio:         ratio,
			Models:        []string{},
			EndpointTypes: []constant.EndpointType{},
		}
		if status, ok := result.Groups[name]; ok {
			item.Status = &status
		}
		groups = append(groups, item)
	}
	for _, pricing := range model.GetPricing() {
		inAllGroups := slices.Contains(pricing.EnableGroup, "all")
		for i := range groups {
			if !inAllGroups && !slices.Contains(pricing.EnableGroup, groups[i].Name) {
				continue
			}
			groups[i].Models = append(groups[i].Models, pricing.ModelName)
			for _, endpointType := range pricing.SupportedEndpointTypes {
				if !slices.Contains(groups[i].EndpointTypes, endpointType) {
					groups[i].EndpointTypes = append(groups[i].EndpointTypes, endpointType)
				}
			}
		}
	}
	for i := range groups {
		// Busiest models first, so a shortened model list stays representative.
		requests := result.Groups[groups[i].Name].ModelRequests
		slices.SortFunc(groups[i].Models, func(a, b string) int {
			return cmp.Or(cmp.Compare(requests[b], requests[a]), cmp.Compare(a, b))
		})
	}
	slices.SortFunc(groups, func(a, b perfGroupStatusItem) int {
		return cmp.Or(cmp.Compare(result.Groups[b.Name].RequestCount, result.Groups[a.Name].RequestCount), cmp.Compare(a.Name, b.Name))
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"summary":       result.Summary,
			"request_count": result.RequestCount,
			"window_start":  result.WindowStart,
			"window_end":    result.WindowEnd,
			"groups":        groups,
		},
	})
}

func GetPerfMetricsSummary(c *gin.Context) {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	// Performance history is keyed by the group used at request time. Do not
	// filter it by the currently configured group ratios: removing or renaming
	// a group must not make existing model metrics disappear from the square.
	result, err := perfmetrics.QuerySummaryAll(hours, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetPerfMetrics(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "model is required",
		})
		return
	}

	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	result, err := perfmetrics.Query(perfmetrics.QueryParams{
		Model: modelName,
		Group: c.Query("group"),
		Hours: hours,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
