package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRSGatewayTaskMigrationPreservesBillingAndUpstreamIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Task{}))
	task := Task{TaskID: "public-task", Platform: "61", Quota: 125, PrivateData: TaskPrivateData{}}
	task.PrivateData.UpstreamTaskID = "private/task"
	task.PrivateData.Key = "stored-key"
	task.PrivateData.BillingContext = &TaskBillingContext{PerCallBilling: true, ModelPrice: 0.25}
	require.NoError(t, db.Create(&task).Error)
	require.NoError(t, migrateRSGatewayTasks(db))
	require.NoError(t, db.First(&task, task.ID).Error)
	assert.Equal(t, "rs-gateway", string(task.Platform))
	assert.Equal(t, "private/task", task.GetUpstreamTaskID())
	assert.Equal(t, "stored-key", task.PrivateData.Key)
	assert.Equal(t, 125, task.Quota)
	require.NotNil(t, task.PrivateData.Execution.TaskPlugin)
	assert.Equal(t, "rs-gateway", task.PrivateData.Execution.TaskPlugin.Key)
	before, err := common.Marshal(task)
	require.NoError(t, err)
	require.NoError(t, migrateRSGatewayTasks(db))
	require.NoError(t, db.First(&task, task.ID).Error)
	after, err := common.Marshal(task)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after))
}
