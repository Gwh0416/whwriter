package store

import (
	"database/sql"
	"time"
	"whwriter/pkg/models"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(user *models.User) error {
	result, err := s.db.Exec(
		"INSERT INTO users (email, username, password_hash) VALUES (?, ?, ?)",
		user.Email, user.Username, user.PasswordHash,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

func (s *UserStore) FindByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, email, username, password_hash, created_at, updated_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserStore) FindByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, email, username, password_hash, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserStore) FindByID(id int64) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(
		"SELECT id, email, username, password_hash, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserStore) SaveVerificationCode(email, code string) error {
	_, err := s.db.Exec(
		"INSERT INTO email_verifications (email, code, expires_at) VALUES (?, ?, ?)",
		email, code, time.Now().Add(10*time.Minute),
	)
	return err
}

func (s *UserStore) VerifyCode(email, code string) (bool, error) {
	var used bool
	var expiresAt time.Time
	err := s.db.QueryRow(
		"SELECT used, expires_at FROM email_verifications WHERE email = ? AND code = ? ORDER BY id DESC LIMIT 1",
		email, code,
	).Scan(&used, &expiresAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if used || time.Now().After(expiresAt) {
		return false, nil
	}
	s.db.Exec("UPDATE email_verifications SET used = TRUE WHERE email = ? AND code = ?", email, code)
	return true, nil
}
