package hailuo

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHailuoDurationAliases(t *testing.T) {
	for _, body := range []string{`{"duration":10}`, `{"duration":"10"}`, `{"seconds":10}`, `{"seconds":"10"}`} {
		t.Run(body, func(t *testing.T) {
			var request relaycommon.TaskSubmitReq
			require.NoError(t, common.Unmarshal([]byte(body), &request))
			payload, err := (&TaskAdaptor{}).convertToRequestPayload(&request, &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "MiniMax-Hailuo-02"}})
			require.NoError(t, err)
			data, err := common.Marshal(payload)
			require.NoError(t, err)
			var result map[string]any
			require.NoError(t, common.Unmarshal(data, &result))
			assert.Equal(t, float64(10), result["duration"])
			assert.NotContains(t, result, "seconds")
		})
	}
}
