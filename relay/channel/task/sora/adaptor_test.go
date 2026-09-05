package sora

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoraBuildRequestBodyReturnsReplayablePassThroughBody(t *testing.T) {
	payload := []byte("opaque-sora-request-body")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/octet-stream")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	replayable, ok := body.(common.ReplayableBody)
	require.True(t, ok)

	sent, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, sent)
	assert.EqualValues(t, len(payload), replayable.Size())

	replayBody, err := replayable.NewReader()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)
}

func TestVideoDurationForwarding(t *testing.T) {
	for _, channelType := range []int{constant.ChannelTypeSora, constant.ChannelTypeRSGateway} {
		for _, form := range []bool{false, true} {
			for _, fields := range []string{`"duration":"6"`, `"seconds":6`, `"duration":"6","seconds":6`} {
				body := []byte(`{"model":"test-video","prompt":"test","extra":"preserved",` + fields + `}`)
				contentType := "application/json"
				if form {
					var values map[string]any
					require.NoError(t, common.Unmarshal(body, &values))
					var buf bytes.Buffer
					writer := multipart.NewWriter(&buf)
					for key, value := range values {
						require.NoError(t, writer.WriteField(key, fmt.Sprint(value)))
					}
					require.NoError(t, writer.Close())
					body, contentType = buf.Bytes(), writer.FormDataContentType()
				}
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(body))
				c.Request.Header.Set("Content-Type", contentType)
				defer common.CleanupBodyStorage(c)
				info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "mapped-video"}, TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
				adaptor := &TaskAdaptor{ChannelType: channelType}
				require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))
				sent, err := adaptor.BuildRequestBody(c, info)
				require.NoError(t, err)
				var result map[string]any
				if form {
					request := httptest.NewRequest(http.MethodPost, "/", sent)
					request.Header.Set("Content-Type", c.GetHeader("Content-Type"))
					require.NoError(t, request.ParseMultipartForm(1<<20))
					result = map[string]any{}
					for key, values := range request.MultipartForm.Value {
						result[key] = values[0]
					}
				} else {
					require.NoError(t, common.DecodeJson(sent, &result))
				}
				assert.Equal(t, "preserved", result["extra"])
				assert.Equal(t, "mapped-video", result["model"])
				if channelType == constant.ChannelTypeSora {
					assert.Equal(t, "6", result["seconds"])
					assert.NotContains(t, result, "duration")
				} else {
					if strings.Contains(fields, "seconds") {
						assert.Equal(t, "6", result["seconds"])
					} else {
						assert.NotContains(t, result, "seconds")
					}
					if strings.Contains(fields, "duration") {
						if form {
							assert.Equal(t, "6", result["duration"])
						} else {
							assert.Equal(t, float64(6), result["duration"])
						}
					} else {
						assert.NotContains(t, result, "duration")
					}
				}
			}
		}
	}
}

func TestRSGatewayResponseRemovesUpstreamVideoURLs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeRSGateway},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
	}
	resp := &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(bytes.NewBufferString(
		`{"id":"video_upstream","status":"queued","metadata":{"url":"https://provider.example/signed.mp4","content_url":"https://provider.example/v1/videos/video_upstream/content"}}`)), Header: make(http.Header)}
	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "video_upstream", upstreamID)
	assert.NotContains(t, string(taskData), "provider.example")
	assert.Contains(t, string(taskData), "task_public")
	assert.NotContains(t, recorder.Body.String(), "provider.example")
	assert.Contains(t, recorder.Body.String(), "task_public")
}
