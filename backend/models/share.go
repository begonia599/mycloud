package models

import "time"

type Share struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code      string     `json:"code" gorm:"type:varchar(20);uniqueIndex;not null"`
	Title     string     `json:"title" gorm:"type:varchar(255)"`
	Password  string     `json:"-" gorm:"type:varchar(255)"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
	Files     []File     `json:"files" gorm:"many2many:share_files;"`
}

type ShareFile struct {
	ShareID string `gorm:"type:uuid;primaryKey"`
	FileID  string `gorm:"type:uuid;primaryKey"`
}

func (ShareFile) TableName() string {
	return "share_files"
}
