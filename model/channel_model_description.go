package model

import "time"

// ChannelModelDescription is an operator-provided description for one concrete
// model exposed by one channel. An empty ChannelID is not allowed so that the
// global model metadata remains the fallback.
type ChannelModelDescription struct {
	Id          int       `json:"id" gorm:"primaryKey"`
	ChannelID   int       `json:"channel_id" gorm:"uniqueIndex:uk_channel_model_description,priority:1;index"`
	ModelName   string    `json:"model_name" gorm:"size:128;uniqueIndex:uk_channel_model_description,priority:2"`
	Description string    `json:"description" gorm:"type:text"`
	Enabled     bool      `json:"enabled" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func GetChannelModelDescriptions() ([]ChannelModelDescription, error) {
	var rows []ChannelModelDescription
	err := DB.Where("enabled = ?", true).Find(&rows).Error
	return rows, err
}

func UpsertChannelModelDescription(row *ChannelModelDescription) error {
	return DB.Where("channel_id = ? AND model_name = ?", row.ChannelID, row.ModelName).
		Assign(map[string]any{"description": row.Description, "enabled": row.Enabled}).
		FirstOrCreate(row).Error
}
