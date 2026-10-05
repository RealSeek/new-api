package perfmetrics

import (
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/model"
)

const (
	groupStatusHours    = 24
	groupStatusCacheTTL = 30 * time.Second
	// groupRecentMinSamples is how many samples a trailing window needs before
	// it is trusted as the group's current health.
	groupRecentMinSamples = 20
)

// groupRecentWindows are the trailing windows, in hours, tried from the
// shortest until one holds enough samples.
var groupRecentWindows = []int{1, 3, 6, 12, 24}

type groupStatusCacheItem struct {
	expiresAt time.Time
	result    GroupStatusResult
}

var (
	groupStatusCacheMu sync.Mutex
	groupStatusCache   = map[string]groupStatusCacheItem{}
)

// QueryGroupStatus aggregates the last 24 hours of samples per group for the
// public status board. Every visitor polls the same board, so results are
// cached briefly per group set.
func QueryGroupStatus(groups []string) (GroupStatusResult, error) {
	now := time.Now()
	sortedGroups := slices.Sorted(slices.Values(groups))
	cacheKey := strings.Join(sortedGroups, "\x00")
	groupStatusCacheMu.Lock()
	cached, ok := groupStatusCache[cacheKey]
	groupStatusCacheMu.Unlock()
	if ok && now.Before(cached.expiresAt) {
		return cached.result, nil
	}

	startTs, endTs := queryWindow(now, groupStatusHours)
	hourly := map[string]map[int64]counters{}
	modelRequests := map[string]map[string]int64{}
	requested := allowedGroupSet(sortedGroups)
	// The existing per-group-set query keeps this path free of new SQL; the
	// short cache bounds the per-group round trips.
	for _, group := range sortedGroups {
		rows, err := model.GetPerfMetricsSummaryBucketsAll(startTs, endTs, []string{group})
		if err != nil {
			return GroupStatusResult{}, err
		}
		for _, row := range rows {
			mergeModelBucket(hourly, group, row.BucketTs-row.BucketTs%3600, counters{
				requestCount:   row.RequestCount,
				successCount:   row.SuccessCount,
				totalLatencyMs: row.TotalLatencyMs,
				outputTokens:   row.OutputTokens,
				generationMs:   row.GenerationMs,
			})
			if modelRequests[group] == nil {
				modelRequests[group] = map[string]int64{}
			}
			modelRequests[group][row.ModelName] += row.RequestCount
		}
	}
	hotBuckets.Range(func(key, value any) bool {
		k := key.(bucketKey)
		if _, ok := requested[k.group]; !ok || k.bucketTs < startTs || k.bucketTs > endTs {
			return true
		}
		snap := value.(*atomicBucket).snapshot()
		if snap.requestCount == 0 {
			return true
		}
		mergeModelBucket(hourly, k.group, k.bucketTs-k.bucketTs%3600, snap)
		if modelRequests[k.group] == nil {
			modelRequests[k.group] = map[string]int64{}
		}
		modelRequests[k.group][k.model] += snap.requestCount
		return true
	})

	currentHour := endTs - endTs%3600
	all := counters{}
	result := GroupStatusResult{WindowStart: startTs, WindowEnd: endTs, Groups: make(map[string]GroupStatus, len(hourly))}
	for group, buckets := range hourly {
		total := counters{}
		series := make([]GroupHourPoint, 0, groupStatusHours)
		for hourTs := startTs; hourTs <= currentHour; hourTs += 3600 {
			value := buckets[hourTs]
			total.add(value)
			series = append(series, GroupHourPoint{
				Ts:           hourTs,
				RequestCount: value.requestCount,
				SuccessRate:  math.Round(successRate(value)*100) / 100,
			})
		}
		if total.requestCount == 0 {
			continue
		}
		all.add(total)

		recent := GroupRecent{}
		for _, hours := range groupRecentWindows {
			window := counters{}
			for _, point := range series[max(len(series)-hours, 0):] {
				window.add(buckets[point.Ts])
			}
			recent = GroupRecent{
				Hours:        hours,
				RequestCount: window.requestCount,
				SuccessRate:  math.Round(successRate(window)*100) / 100,
			}
			if window.requestCount >= groupRecentMinSamples {
				break
			}
		}
		result.Groups[group] = GroupStatus{
			RequestCount:  total.requestCount,
			SuccessRate:   math.Round(successRate(total)*100) / 100,
			AvgLatencyMs:  avg(total.totalLatencyMs, total.requestCount),
			AvgTps:        math.Round(avgTps(total)*100) / 100,
			Recent:        recent,
			Series:        series,
			ModelRequests: modelRequests[group],
		}
	}
	result.Summary = summarize(all)
	result.RequestCount = all.requestCount

	groupStatusCacheMu.Lock()
	groupStatusCache[cacheKey] = groupStatusCacheItem{expiresAt: now.Add(groupStatusCacheTTL), result: result}
	groupStatusCacheMu.Unlock()
	return result, nil
}

func (c *counters) add(value counters) {
	c.requestCount += value.requestCount
	c.successCount += value.successCount
	c.totalLatencyMs += value.totalLatencyMs
	c.ttftSumMs += value.ttftSumMs
	c.ttftCount += value.ttftCount
	c.outputTokens += value.outputTokens
	c.generationMs += value.generationMs
}
