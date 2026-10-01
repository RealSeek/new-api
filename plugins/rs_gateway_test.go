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
	value, err := plugin.Engine.Call(t.Context(), "buildSubmitRequest", ctx)
	require.NoError(t, err)
	request := value.(map[string]any)
	assert.Equal(t, "https://gateway.example/v1/videos", request["url"])
	assert.Equal(t, "mapped-video", request["body"].(map[string]any)["model"])
	assert.Equal(t, body["parameters"], request["body"].(map[string]any)["parameters"])
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

func TestRSGatewayRejectsFractionalDurationBeforeBilling(t *testing.T) {
	source, err := Source("rs-gateway")
	require.NoError(t, err)
	plugin, err := jsplugin.CompilePlugin(source, jsplugin.Options{})
	require.NoError(t, err)
	_, err = plugin.Engine.Call(t.Context(), "buildSubmitRequest", map[string]any{"requestBody": map[string]any{"seconds": 8.5}, "upstreamModel": "video", "baseUrl": "https://gateway.example"})
	require.Error(t, err)
}
