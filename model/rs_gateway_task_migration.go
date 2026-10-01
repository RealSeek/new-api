package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

func migrateRSGatewayTasks(db *gorm.DB) error {
	var tasks []Task
	return db.Where("platform = ?", "61").FindInBatches(&tasks, 100, func(tx *gorm.DB, _ int) error {
		for _, task := range tasks {
			if task.PrivateData.Execution == nil {
				task.PrivateData.Execution = &TaskExecutionSnapshot{}
			}
			task.PrivateData.Execution.TaskPlugin = &TaskPluginSnapshot{
				Key: "rs-gateway", Name: "RS Gateway", Version: "1.0.0", APIVersion: 1,
				Author: &TaskPluginAuthorSnapshot{Name: "RealSeek"},
			}
			if err := tx.Model(&Task{}).Where("id = ? AND platform = ?", task.ID, "61").Updates(map[string]any{
				"platform": constant.TaskPlatform("rs-gateway"), "private_data": task.PrivateData,
			}).Error; err != nil {
				return fmt.Errorf("migrate gateway task %s: %w", task.TaskID, err)
			}
		}
		return nil
	}).Error
}
