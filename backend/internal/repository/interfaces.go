package repository

import "whwriter/backend/internal/model"

type UserRepository interface {
	Create(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByUsername(username string) (*model.User, error)
	FindByID(id uint) (*model.User, error)
	SaveVerificationCode(email, code string) error
	VerifyCode(email, code string) (bool, error)
	UpdatePassword(userID uint, passwordHash string) error
	ListUsers(page, pageSize int) ([]model.User, int64, error)
	GetStats() (*DashboardStats, error)
}

type DashboardStats struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveBooks   int64 `json:"active_books"`
	TotalChapters int64 `json:"total_chapters"`
}

type GenreRepository interface {
	ListBuiltin() ([]model.Genre, error)
	ListByUser(userID uint) ([]model.Genre, error)
}

type PlatformRepository interface {
	List() ([]model.Platform, error)
}

type LLMConfigRepository interface {
	List() ([]model.LLMConfig, error)
}

type BookRepository interface {
	Create(book *model.Book) error
	ListByUser(userID uint) ([]model.Book, error)
}
