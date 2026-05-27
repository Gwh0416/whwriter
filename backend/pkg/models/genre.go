package models

import "time"

type Genre struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	UserID          uint      `json:"user_id" gorm:"uniqueIndex:idx_genre_user_name;default:0"`
	Name            string    `json:"name" gorm:"uniqueIndex:idx_genre_user_name;size:64;not null"`
	ProfileMarkdown string    `json:"profile_markdown" gorm:"type:longtext"`
	SortOrder       int       `json:"sort_order" gorm:"default:0"`
	IsActive        bool      `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Genre) TableName() string {
	return "genres"
}

type Platform struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Name       string    `json:"name" gorm:"uniqueIndex;size:64;not null"`
	StyleGuide string    `json:"style_guide" gorm:"type:longtext"`
	SortOrder  int       `json:"sort_order" gorm:"default:0"`
	IsActive   bool      `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Platform) TableName() string {
	return "platforms"
}
