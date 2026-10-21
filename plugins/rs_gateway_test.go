package plugins

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSGatewayNativeVideoContract(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)

	t.Run("native JSON preserves extensions and model mapping", func(t *testing.T) {
		body := map[string]any{
			"model": "public-video", "prompt": "a wave", "seconds": "8", "size": "1280x720",
			"input_reference": map[string]any{"image_url": "https://cdn.example/reference.png"},
			"parameters":      map[string]any{"seed": 0, "watermark": false},
			"metadata":        "opaque metadata; not a JSON document",
		}
		decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
			"model": "public-video", "body": map[string]any{"kind": "json", "value": body},
		})
		require.NoError(t, err)
		assert.Equal(t, body, decoded.(map[string]any)["requestBody"])
		value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{
			"requestBody": body, "upstreamModel": "mapped-video", "baseUrl": "https://gateway.example/", "apiKey": "channel-key",
			"requestHeaders": map[string]any{"Idempotency-Key": "art-video-job-1"},
		})
		require.NoError(t, err)
		request := value.(map[string]any)
		assert.Equal(t, "https://gateway.example/v1/videos", request["url"])
		forwarded := request["body"].(map[string]any)
		assert.Equal(t, "mapped-video", forwarded["model"])
		assert.Equal(t, "public-video", body["model"])
		assert.Equal(t, body["input_reference"], forwarded["input_reference"])
		assert.Equal(t, body["parameters"], forwarded["parameters"])
		assert.Equal(t, body["metadata"], forwarded["metadata"])
		assert.Equal(t, body["prompt"], forwarded["prompt"])
		assert.NotContains(t, forwarded, "content")
		assert.Equal(t, "art-video-job-1", request["headers"].(map[string]any)["Idempotency-Key"])
	})

	t.Run("multipart preserves fields and file references", func(t *testing.T) {
		decoded, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "decodeRequest"}, map[string]any{
			"model": "seedance-2.5", "body": map[string]any{"kind": "multipart", "fields": map[string]any{
				"model": []string{"seedance-2.5"}, "prompt": []string{"from files"}, "seconds": []string{"8"}, "size": []string{"1280x720"},
				"custom_field": []string{"first", "second"}, "metadata": []string{`{"enabled":false,"seed":0}`},
			}},
		})
		require.NoError(t, err)
		value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{
			"requestBody": decoded.(map[string]any)["requestBody"], "upstreamModel": "mapped-video", "baseUrl": "https://gateway.example", "apiKey": "channel-key",
			"requestHeaders": map[string]any{"Content-Type": "multipart/form-data; boundary=original-boundary"},
			"files":          []any{map[string]any{"ref": "request_file:input_reference", "field": "input_reference", "filename": "reference.png"}},
		})
		require.NoError(t, err)
		request := value.(map[string]any)
		assert.Equal(t, "multipart", request["bodyType"])
		assert.NotContains(t, request, "body")
		assert.NotContains(t, request["headers"], "Content-Type", "the host creates the actual multipart boundary")
		assert.Contains(t, request["parts"], map[string]any{"name": "custom_field", "value": "first"})
		assert.Contains(t, request["parts"], map[string]any{"name": "custom_field", "value": "second"})
		assert.Contains(t, request["parts"], map[string]any{"name": "metadata", "value": `{"enabled":false,"seed":0}`})
		assert.Contains(t, request["parts"], map[string]any{"name": "input_reference", "fileRef": "request_file:input_reference", "filename": "reference.png"})
	})

	t.Run("billing duration cannot differ from the native request", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			body    map[string]any
			message string
		}{
			{"removed custom protocol", map[string]any{"contract_version": "video-v1", "prompt": "a wave"}, "video-v1 is no longer supported"},
			{"fractional", map[string]any{"prompt": "a wave", "seconds": 8.5}, "duration must be between"},
			{"oversized", map[string]any{"prompt": "a wave", "seconds": "3601"}, "duration must be between"},
			{"unsupported automatic duration", map[string]any{"prompt": "a wave", "seconds": "-1"}, "automatic duration is not declared"},
			{"other negative", map[string]any{"prompt": "a wave", "seconds": "-2"}, "duration must be between"},
			{"boolean", map[string]any{"prompt": "a wave", "seconds": true}, "duration must be between"},
			{"hex string", map[string]any{"prompt": "a wave", "seconds": "0x8"}, "duration must be between"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"requestBody": tc.body, "upstreamModel": "video", "baseUrl": "https://gateway.example"})
				require.ErrorContains(t, err, tc.message)
				if tc.name != "removed custom protocol" {
					_, err = plugin.Engine.Call(t.Context(), "extractUsage", map[string]any{"requestBody": tc.body, "upstreamModel": "video"})
					require.ErrorContains(t, err, tc.message)
				}
			})
		}
	})

	t.Run("Seedance automatic duration preserves the request and reserves its model maximum", func(t *testing.T) {
		for _, tc := range []struct {
			model string
			limit int
		}{
			{"seedance-2.0", 15}, {"doubao-seedance-2-5-260628", 30},
		} {
			ctx := map[string]any{"model": tc.model, "upstreamModel": tc.model, "baseUrl": "https://gateway.example", "requestBody": map[string]any{"prompt": "@参考图0 moves", "seconds": "-1", "resolution": "720p", "aspect_ratio": "adaptive", "videos": []string{"https://cdn.example/input.mp4"}}}
			request, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
			require.NoError(t, err)
			assert.Equal(t, "-1", request.(map[string]any)["body"].(map[string]any)["seconds"])
			usage, err := plugin.Engine.Call(t.Context(), "extractUsage", ctx)
			require.NoError(t, err)
			facts := usage.(map[string]any)
			assert.EqualValues(t, tc.limit, facts["seconds"])
			assert.EqualValues(t, tc.limit*1280*720*24/1024, facts["tokens"])
			assert.Equal(t, true, facts["video_input"])
		}
	})

	t.Run("completion preserves extensions with local media access", func(t *testing.T) {
		localURL := "https://newapi.example/v1/tasks/public-task/artifacts/video/content?access=local-access"
		body := map[string]any{
			"id": "private-task", "status": "completed", "seconds": "8", "size": "1280x720",
			"usage":    map[string]any{"completion_tokens": 123456, "seconds": 8},
			"metadata": map[string]any{"custom": map[string]any{"seed": 0, "enabled": false}, "url": "https://gateway.example/video?access=private"},
			"result":   map[string]any{"url": "https://cdn.example/signed.mp4?signature=private", "last_frame_url": "https://cdn.example/last.png", "format": "mp4"},
		}
		value, err := plugin.Engine.CallPath(t.Context(), "protocols", []string{"openai_video", "render"},
			map[string]any{"model": "public-video", "artifacts": map[string]any{"video": map[string]any{"url": localURL}}},
			map[string]any{"task_id": "public-task", "status": "SUCCESS", "data": body})
		require.NoError(t, err)
		result := value.(map[string]any)
		assert.Equal(t, "public-task", result["id"])
		assert.NotContains(t, result, "task_id")
		assert.Equal(t, "public-video", result["model"])
		usage := result["usage"].(map[string]any)
		assert.EqualValues(t, 123456, usage["completion_tokens"])
		assert.EqualValues(t, 8, usage["seconds"])
		assert.Equal(t, body["seconds"], result["seconds"])
		assert.Equal(t, body["size"], result["size"])
		assert.Equal(t, localURL, result["result"].(map[string]any)["url"])
		assert.Equal(t, "mp4", result["result"].(map[string]any)["format"])
		assert.NotContains(t, result["result"], "last_frame_url")
		custom := result["metadata"].(map[string]any)["custom"].(map[string]any)
		assert.EqualValues(t, 0, custom["seed"])
		assert.Equal(t, false, custom["enabled"])
		assert.Equal(t, localURL, result["metadata"].(map[string]any)["url"])
		assert.Equal(t, "private-task", body["id"], "rendering must not mutate the upstream snapshot")
	})

	t.Run("content always uses the ownership checked channel task", func(t *testing.T) {
		for _, method := range []string{"GET", "HEAD"} {
			value, err := plugin.Engine.Call(t.Context(), "buildContentRequest", map[string]any{
				"artifactKey": "video", "upstreamTaskId": "private/task", "baseUrl": "https://gateway.example", "apiKey": "channel-secret",
				"model": "seedance-2.5", "data": map[string]any{"result": map[string]any{"url": "https://cdn.example/signed.mp4"}},
				"clientRequest": map[string]any{"method": method},
			})
			require.NoError(t, err)
			request := value.(map[string]any)
			assert.Equal(t, "https://gateway.example/v1/videos/private%2Ftask/content", request["url"])
			assert.Equal(t, method, request["method"])
			assert.Equal(t, "Bearer channel-secret", request["headers"].(map[string]any)["Authorization"])
			assert.NotContains(t, request, "credentialless")
		}
	})

	t.Run("gateway failure reason remains visible", func(t *testing.T) {
		message := "请求内容未通过安全检查，请调整描述后重试"
		value, err := plugin.Engine.Call(t.Context(), "parseTaskResult", map[string]any{}, map[string]any{"status": "failed", "error": map[string]any{"message": message}})
		require.NoError(t, err)
		assert.Equal(t, "FAILURE", value.(map[string]any)["status"])
		assert.Equal(t, message, value.(map[string]any)["reason"])
	})

	t.Run("callback remains local", func(t *testing.T) {
		body := map[string]any{"model": "video", "prompt": "a wave", "seconds": "8", "callback_url": "https://hooks.example/video"}
		value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"requestBody": body, "upstreamModel": "video", "baseUrl": "https://gateway.example"})
		require.NoError(t, err)
		assert.NotContains(t, value.(map[string]any)["body"], "callback_url")
		value, err = plugin.Engine.Call(t.Context(), "parseSubmitResponse", map[string]any{"requestBody": body}, map[string]any{"body": map[string]any{"id": "gateway-task"}})
		require.NoError(t, err)
		state := value.(map[string]any)["state"].(map[string]any)
		assert.Equal(t, body["callback_url"], state["callback"].(map[string]any)["url"])
	})
}

func TestRSGatewayNativeVideoBillingFacts(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	value, err := plugin.Engine.Call(t.Context(), "extractUsage", map[string]any{
		"upstreamModel": "[NSFW]seedance-2.5", "requestBody": map[string]any{"prompt": "a wave", "seconds": "5", "size": "1920x1080"},
	})
	require.NoError(t, err)
	facts := value.(map[string]any)
	assert.EqualValues(t, 1, facts["requests"])
	assert.EqualValues(t, 5, facts["seconds"])
	assert.Equal(t, "1080p", facts["resolution"])
	assert.EqualValues(t, 243000, facts["tokens"])
	for _, tc := range []struct {
		name string
		body map[string]any
		want map[string]any
	}{
		{"measured usage wins", map[string]any{"seconds": "5", "size": "1280x720", "usage": map[string]any{"completion_tokens": 123456, "seconds": 8, "size": "1920x1080"}}, map[string]any{"tokens": 123456, "seconds": 8, "resolution": "1080p"}},
		{"explicit zero is real usage", map[string]any{"seconds": "5", "usage": map[string]any{"completion_tokens": 0, "seconds": 0}}, map[string]any{"tokens": 0, "seconds": 0}},
		{"native snapshot fallback", map[string]any{"seconds": "8", "size": "1280x720"}, map[string]any{"seconds": float64(8), "resolution": "720p"}},
		{"missing usage keeps reservation", map[string]any{}, map[string]any{}},
		{"automatic duration is not measured usage", map[string]any{"seconds": "-1", "resolution": "720p"}, map[string]any{}},
		{"negative measured duration is not billable", map[string]any{"usage": map[string]any{"seconds": -1}}, map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := plugin.Engine.Call(t.Context(), "extractUsageOnComplete", map[string]any{}, map[string]any{"status": "SUCCESS"}, tc.body)
			require.NoError(t, err)
			facts := value.(map[string]any)
			assert.Len(t, facts, len(tc.want))
			for key, expected := range tc.want {
				assert.EqualValues(t, expected, facts[key], key)
			}
		})
	}
}

func TestRSGatewayHHCodingAutomaticDuration(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	for _, name := range []string{"MiniMaxH3", "[c]MiniMaxH3", "MiniMaxH3-480p", "MiniMaxH3-720p-sec", "[c]MiniMaxH3-2k-sec", "MiniMaxH3-2k-pro-sec"} {
		t.Run(name, func(t *testing.T) {
			ctx := map[string]any{"upstreamModel": name, "baseUrl": "https://gateway.example", "requestBody": map[string]any{"prompt": "a wave", "seconds": "-1", "resolution": "720p"}}
			request, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
			require.NoError(t, err)
			assert.Equal(t, "-1", request.(map[string]any)["body"].(map[string]any)["seconds"])
			usage, err := plugin.Engine.Call(t.Context(), "extractUsage", ctx)
			require.NoError(t, err)
			facts := usage.(map[string]any)
			assert.EqualValues(t, 15, facts["seconds"])
			assert.NotContains(t, facts, "tokens", "an HHCoding reservation cannot invent token usage")
		})
	}
	_, err = plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"upstreamModel": "MiniMax-H3", "requestBody": map[string]any{"prompt": "a wave", "seconds": -1}})
	require.ErrorContains(t, err, "automatic duration is not declared")
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
	channelPlugin, found := registry.Generation().GetByChannelType(61)
	require.True(t, found)
	assert.Same(t, gatewayPlugin, channelPlugin)
	candidates := registry.Generation().LookupEndpointCandidates("POST", "/v1/videos", "MiniMax-H3")
	require.Len(t, candidates, 2)
	assert.Same(t, nativePlugin, candidates[0].Plugin)
	assert.Same(t, gatewayPlugin, candidates[1].Plugin)
}
