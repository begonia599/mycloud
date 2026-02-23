package models

import "time"

type File struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name       string    `json:"name" gorm:"type:varchar(255);not null"`
	StoredName string    `json:"stored_name" gorm:"type:varchar(255);not null"`
	Size       int64     `json:"size" gorm:"not null"`
	MimeType   string    `json:"mime_type" gorm:"type:varchar(100)"`
	CreatedAt  time.Time `json:"created_at"`
}
