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
	nativeSource, err := Source("hailuo")
	require.NoError(t, err)
	registry := jsplugin.NewRegistry()
	nativePlugin, err := registry.RegisterFactory(nativeSource, jsplugin.Options{})
	require.NoError(t, err)

	gatewaySource, err := Source("rs-gateway")
	require.NoError(t, err)
	gatewayPlugin, err := registry.RegisterFactory(gatewaySource, jsplugin.Options{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"seedance-2.0", "seedance-2.5", "doubao-seedance-2-0-260128", "doubao-seedance-2-5-260628", "[c]seedance-2.0", "[c]seedance-2.5",
		"MiniMax-H3", "[c]MiniMaxH3", "grok-imagine-video-1.5",
	}, gatewayPlugin.Meta.Models)
	channelPlugin, found := registry.Generation().GetByChannelType(61)
	require.True(t, found)
	assert.Same(t, gatewayPlugin, channelPlugin)
	candidates := registry.Generation().LookupEndpointCandidates("POST", "/v1/videos", "MiniMax-H3")
	require.Len(t, candidates, 2)
	assert.Same(t, nativePlugin, candidates[0].Plugin)
	assert.Same(t, gatewayPlugin, candidates[1].Plugin)
	for _, model := range []string{"seedance-2.0", "seedance-2.5", "doubao-seedance-2-0-260128", "doubao-seedance-2-5-260628", "[c]seedance-2.0", "[c]MiniMaxH3", "grok-imagine-video-1.5"} {
		candidates := registry.Generation().LookupEndpointCandidates("POST", "/v1/videos", model)
		require.NotEmpty(t, candidates, model)
		assert.Same(t, gatewayPlugin, candidates[len(candidates)-1].Plugin)
	}
}

func TestRSGatewayMapsFullSeedanceResolutionWithoutChangingPublicRequest(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	for _, tc := range []struct {
		model, resolution, want string
	}{
		{"seedance-2.0", "4k", "seedance-2.0"},
		{"seedance-2.5", "720p", "seedance-2.5"},
		{"doubao-seedance-2-0-260128", "4k", "doubao-seedance-2-0-260128"},
		{"doubao-seedance-2-5-260628", "720p", "doubao-seedance-2-5-260628"},
		{"[c]seedance-2.0", "720p", "[c]seedance-2.0"},
	} {
		request := map[string]any{
			"contract_version": "video-v1", "model": tc.model, "prompt": "a wave",
			"duration": 5, "resolution": tc.resolution, "ratio": "16:9",
		}
		value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{
			"model": tc.model, "upstreamModel": tc.model, "requestBody": request,
			"baseUrl": "https://gateway.example", "apiKey": "channel-key",
		})
		require.NoError(t, err)
		body := value.(map[string]any)["body"].(map[string]any)
		assert.Equal(t, tc.want, body["model"])
		assert.Equal(t, tc.resolution, body["resolution"])
		assert.Equal(t, tc.model, request["model"])
	}
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
		"requestBody":   map[string]any{"model": "[c]seedance-2.5", "duration": 30, "resolution": "720p"},
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
			name:        "official Seedance model name",
			model:       "doubao-seedance-2-5-260628",
			body:        map[string]any{"model": "doubao-seedance-2-5-260628", "content": []any{map[string]any{"type": "text", "text": "a wave"}}, "duration": -1, "resolution": "720p", "ratio": "16:9"},
			wantSeconds: 30,
		},
		{
			name:        "official MiniMax H3 body",
			model:       "MiniMax-H3",
			body:        map[string]any{"model": "MiniMax-H3", "content": []any{map[string]any{"type": "text", "text": "a wave"}}, "duration": 5, "resolution": "768P", "ratio": "16:9"},
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

func TestRSGatewayVideoV1UsesOneRequestShapeForVideoModels(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)

	for _, tc := range []struct {
		name       string
		model      string
		request    map[string]any
		wantMedia  map[string]any
		wantOption map[string]any
	}{
		{
			name:  "Seedance reference video and explicit options",
			model: "seedance-2.5",
			request: map[string]any{
				"contract_version": "video-v1", "model": "seedance-2.5", "prompt": "change the background",
				"duration": 10, "resolution": "720p", "ratio": "adaptive",
				"references": []any{map[string]any{"type": "video", "role": "reference_video", "source": "https://cdn.example/clip.mp4"}},
				"options":    map[string]any{"generate_audio": false, "priority": 0, "video_format": "mp4"},
			},
			wantMedia:  map[string]any{"type": "video_url", "role": "reference_video", "video_url": map[string]any{"url": "https://cdn.example/clip.mp4"}},
			wantOption: map[string]any{"generate_audio": false, "priority": 0, "output_format": "mp4"},
		},
		{
			name:  "MiniMax reference image",
			model: "MiniMax-H3",
			request: map[string]any{
				"contract_version": "video-v1", "model": "MiniMax-H3", "prompt": "animate the portrait",
				"duration": 5, "resolution": "768p", "ratio": "16:9",
				"references": []any{map[string]any{"type": "image", "role": "reference_image", "source": "https://cdn.example/portrait.png"}},
			},
			wantMedia: map[string]any{"type": "image_url", "role": "reference_image", "image_url": map[string]any{"url": "https://cdn.example/portrait.png"}},
		},
		{
			name:  "Grok text only",
			model: "grok-imagine-video-1.5",
			request: map[string]any{
				"contract_version": "video-v1", "model": "grok-imagine-video-1.5", "prompt": "a red paper boat",
				"duration": 5, "resolution": "720p", "ratio": "16:9",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"},
				map[string]any{"model": tc.model, "body": map[string]any{"kind": "json", "value": tc.request}})
			require.NoError(t, err)
			requestBody := decoded.(map[string]any)["requestBody"].(map[string]any)
			assert.NotContains(t, requestBody, "contract_version")
			assert.NotContains(t, requestBody, "references")
			assert.NotContains(t, requestBody, "options")
			content := requestBody["content"].([]any)
			assert.Equal(t, map[string]any{"type": "text", "text": tc.request["prompt"]}, content[0])
			if tc.wantMedia == nil {
				assert.Len(t, content, 1)
			} else {
				assert.Equal(t, tc.wantMedia, content[1])
			}

			value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{
				"upstreamModel": tc.model, "requestBody": tc.request, "baseUrl": "https://gateway.example",
			})
			require.NoError(t, err)
			forwarded := value.(map[string]any)["body"].(map[string]any)
			assert.Equal(t, requestBody["content"], forwarded["content"])
			assert.EqualValues(t, tc.request["duration"], forwarded["duration"])
			for key, expected := range tc.wantOption {
				assert.EqualValues(t, expected, forwarded[key])
			}
		})
	}
	usage, err := plugin.Engine.Call(t.Context(), "extractUsage", map[string]any{
		"upstreamModel": "MiniMax-H3",
		"requestBody": map[string]any{
			"duration":   5,
			"references": []any{map[string]any{"type": "video", "role": "reference_video", "source": "https://cdn.example/clip.mp4"}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, true, usage.(map[string]any)["video_input"])
	rendered, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "render"},
		map[string]any{}, map[string]any{
			"task_id": "public-task", "status": "SUCCESS",
			"properties": map[string]any{"origin_model_name": "seedance-2.5"},
			"data":       map[string]any{"id": "private-task", "status": "succeeded", "metadata": map[string]any{"last_frame_url": "https://cdn.example/last.png"}},
		})
	require.NoError(t, err)
	result := rendered.(map[string]any)
	assert.Equal(t, "public-task", result["id"])
	assert.Equal(t, "completed", result["status"])
	assert.Equal(t, "/v1/videos/public-task/content", result["result"].(map[string]any)["url"])
	assert.Equal(t, "https://cdn.example/last.png", result["result"].(map[string]any)["last_frame_url"])
}

func TestRSGatewayVideoV1RejectsUnsupportedInput(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, extra string
		request     map[string]any
	}{
		{"mixed media formats", "content cannot be combined", map[string]any{"prompt": "a wave", "content": []any{}, "references": []any{}}},
		{"invalid callback", "callback_url must be a valid HTTPS URL", map[string]any{"prompt": "a wave", "options": map[string]any{"callback_url": "http://example.com/hook"}}},
		{"private asset", "reference source must be", map[string]any{"prompt": "a wave", "references": []any{map[string]any{"type": "image", "role": "reference_image", "source": "asset://private"}}}},
		{"fixed Seedance duration", "requires duration 30 seconds", map[string]any{"model": "[c]seedance-2.5", "prompt": "a wave", "duration": 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{
				"upstreamModel": "seedance-2.0", "requestBody": tc.request, "baseUrl": "https://gateway.example",
			})
			require.ErrorContains(t, err, tc.extra)
		})
	}
}

func TestRSGatewayVideoV1MultipartUsesTheCanonicalContentShape(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
		"model": "seedance-2.0",
		"body": map[string]any{
			"kind":   "multipart",
			"fields": map[string]any{"prompt": []string{"from files"}, "duration": []string{"4"}, "resolution": []string{"480p"}, "ratio": []string{"16:9"}},
			"files": []any{
				map[string]any{"ref": "request_file:first_frame#0", "field": "first_frame", "mimeType": "image/png", "size": 3},
				map[string]any{"ref": "request_file:last_frame#0", "field": "last_frame", "mimeType": "image/png", "size": 3},
			},
		},
	})
	require.NoError(t, err)
	requestBody := decoded.(map[string]any)["requestBody"].(map[string]any)
	value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{
		"model": "seedance-2.0", "upstreamModel": "seedance-2.0", "requestBody": requestBody,
		"files": []any{
			map[string]any{"ref": "request_file:first_frame#0", "field": "first_frame", "mimeType": "image/png", "size": 3},
			map[string]any{"ref": "request_file:last_frame#0", "field": "last_frame", "mimeType": "image/png", "size": 3},
		}, "baseUrl": "https://gateway.example",
	})
	require.NoError(t, err)
	body := value.(map[string]any)["body"].(map[string]any)
	content := body["content"].([]any)
	assert.Equal(t, "first_frame", content[1].(map[string]any)["role"])
	assert.Equal(t, "last_frame", content[2].(map[string]any)["role"])
	assert.Equal(t, "dataUrl", content[1].(map[string]any)["image_url"].(map[string]any)["url"].(map[string]any)["encoding"])
}

func TestRSGatewayVideoCallbackIsStoredButNeverForwardedUpstream(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	request := map[string]any{"contract_version": "video-v1", "model": "seedance-2.0", "prompt": "a wave", "duration": 4,
		"options": map[string]any{"callback_url": "https://hooks.example/video"}}
	value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"requestBody": request, "upstreamModel": "seedance-2.0", "baseUrl": "https://gateway.example"})
	require.NoError(t, err)
	assert.NotContains(t, value.(map[string]any)["body"].(map[string]any), "callback_url")
	parsed, err := plugin.Engine.Call(t.Context(), "parseSubmitResponse", map[string]any{"requestBody": map[string]any{"callback_url": "https://hooks.example/video"}}, map[string]any{"body": map[string]any{"id": "upstream-task"}})
	require.NoError(t, err)
	state := parsed.(map[string]any)["state"].(map[string]any)
	assert.Equal(t, "https://hooks.example/video", state["callback"].(map[string]any)["url"])
}
