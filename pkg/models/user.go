package models

import "time"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:255;not null"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	Role         Role      `json:"role" gorm:"size:16;not null;default:user"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EmailVerification struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Email     string    `json:"email" gorm:"size:255;not null;index:idx_email_code"`
	Code      string    `json:"code" gorm:"size:6;not null;index:idx_email_code"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=2,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Code     string `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type SendCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
}
