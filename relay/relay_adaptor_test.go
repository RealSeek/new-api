package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorSupportsRSGatewayVideoTasks(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform("61"))
	require.NotNil(t, adaptor)
	var _ channel.TaskAdaptor = adaptor
}
