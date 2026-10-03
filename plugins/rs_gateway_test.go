package plugins

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSGatewayPluginPreservesArbitraryVideoRequest(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	registry := jsplugin.NewRegistry()
	plugin, err := registry.RegisterFactory(source, jsplugin.Options{})
	require.NoError(t, err)
	body := map[string]any{"model": "custom-video", "prompt": "a wave", "seconds": 5, "parameters": map[string]any{"seed": 0, "watermark": false}}
	ctx := map[string]any{"requestBody": body, "upstreamModel": "mapped-video", "baseUrl": "https://gateway.example/", "apiKey": "channel-key"}
	ctx["requestHeaders"] = map[string]any{"Idempotency-Key": "art-video-job-1"}
	value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
	require.NoError(t, err)
	request := value.(map[string]any)
	assert.Equal(t, "https://gateway.example/v1/videos", request["url"])
	assert.Equal(t, "mapped-video", request["body"].(map[string]any)["model"])
	assert.Equal(t, body["parameters"], request["body"].(map[string]any)["parameters"])
	assert.Equal(t, "art-video-job-1", request["headers"].(map[string]any)["Idempotency-Key"])
	value, err = plugin.Engine.Call(t.Context(), "parseTaskResult", ctx, map[string]any{"id": "task-1", "status": "running"})
	require.NoError(t, err)
	assert.Equal(t, "IN_PROGRESS", value.(map[string]any)["status"])
	value, err = plugin.Engine.Call(t.Context(), "parseSubmitResponse", ctx, map[string]any{"statusCode": 202, "body": map[string]any{"id": "private/task", "status": "queued"}})
	require.NoError(t, err)
	assert.Equal(t, "private/task", value.(map[string]any)["taskId"])
	ctx["artifactKey"], ctx["upstreamTaskId"], ctx["clientRequest"] = "video", "private/task", map[string]any{"method": "GET"}
	value, err = plugin.Engine.Call(t.Context(), "buildContentRequest", ctx)
	require.NoError(t, err)
	assert.Equal(t, "https://gateway.example/v1/videos/private%2Ftask/content", value.(map[string]any)["url"])
	body["metadata"] = map[string]any{"duration": 3601}
	_, err = plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
	require.Error(t, err)
}

func TestRSGatewayClaimsMiniMaxH3VideoEndpoint(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	registry := jsplugin.NewRegistry()
	plugin, err := registry.RegisterFactory(source, jsplugin.Options{})
	require.NoError(t, err)

	assert.Equal(t, []string{"MiniMax-H3"}, plugin.Meta.Models)
	binding, found := registry.Generation().LookupEndpoint("POST", "/v1/videos", "MiniMax-H3")
	require.True(t, found)
	assert.Same(t, plugin, binding.Plugin)
}

func TestRSGatewayRejectsFractionalDurationBeforeBilling(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	_, err = plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"requestBody": map[string]any{"seconds": 8.5}, "upstreamModel": "video", "baseUrl": "https://gateway.example"})
	require.Error(t, err)
}

func TestRSGatewayReportsOneRequestForPerVideoPricing(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	ctx := map[string]any{
		"upstreamModel": "[c]seedance-2.5",
		"requestBody": map[string]any{"model": "[c]seedance-2.5", "duration": 30, "resolution": "720p"},
	}
	value, err := plugin.Engine.Call(t.Context(), "extractUsage", ctx)
	require.NoError(t, err)
	assert.EqualValues(t, 1, value.(map[string]any)["requests"])
	assert.EqualValues(t, 30, value.(map[string]any)["seconds"])
	_, examples := plugin.Meta.UsageForModel("[c]seedance-2.5")
	require.NotEmpty(t, examples)
	assert.EqualValues(t, 1, examples[0].Facts["requests"])
}

func TestRSGatewayOfficialSeedanceAndMiniMaxRequestsKeepRequestUsage(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, model string
		body        map[string]any
		wantSeconds float64
	}{
		{
			name: "official Seedance model name",
			model: "doubao-seedance-2-5-260628",
			body: map[string]any{"model": "doubao-seedance-2-5-260628", "content": []any{map[string]any{"type": "text", "text": "a wave"}}, "duration": -1, "resolution": "720p", "ratio": "16:9"},
			wantSeconds: 30,
		},
		{
			name: "official MiniMax H3 body",
			model: "MiniMax-H3",
			body: map[string]any{"model": "MiniMax-H3", "content": []any{map[string]any{"type": "text", "text": "a wave"}}, "duration": 5, "resolution": "768P", "ratio": "16:9"},
			wantSeconds: 5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := map[string]any{"upstreamModel": tc.model, "requestBody": tc.body, "baseUrl": "https://gateway.example"}
			value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
			require.NoError(t, err)
			request := value.(map[string]any)
			forwarded := request["body"].(map[string]any)
			assert.Equal(t, tc.model, forwarded["model"])
			assert.Equal(t, tc.body["content"], forwarded["content"])
			assert.EqualValues(t, tc.body["duration"], forwarded["duration"])
			assert.Equal(t, tc.body["resolution"], forwarded["resolution"])
			assert.Equal(t, tc.body["ratio"], forwarded["ratio"])
			value, err = plugin.Engine.Call(t.Context(), "extractUsage", ctx)
			require.NoError(t, err)
			usage := value.(map[string]any)
			assert.EqualValues(t, 1, usage["requests"])
			assert.EqualValues(t, tc.wantSeconds, usage["seconds"])
			value, err = plugin.Engine.Call(t.Context(), "extractUsageOnComplete", ctx, map[string]any{"status": "SUCCESS"}, map[string]any{"usage": map[string]any{"output_seconds": tc.wantSeconds}})
			require.NoError(t, err)
			assert.EqualValues(t, tc.wantSeconds, value.(map[string]any)["seconds"])
		})
	}
}
