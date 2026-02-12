package database

import (
	"errors"

	"gorm.io/gorm"
)

type AuthUserStore struct {
	db *gorm.DB
}

func NewAuthUserStore(db *gorm.DB) *AuthUserStore {
	return &AuthUserStore{db: db}
}

func (s *AuthUserStore) Create(usernameNorm, username, passwordHash, playerID string) error {
	if s == nil || s.db == nil {
		return errors.New("auth user store not initialized")
	}
	model := AuthUserModel{
		UsernameNorm: usernameNorm,
		Username:     username,
		PasswordHash: passwordHash,
		PlayerID:     playerID,
	}
	return s.db.Create(&model).Error
}

func (s *AuthUserStore) GetByUsernameNorm(usernameNorm string) (string, string, string, error) {
	if s == nil || s.db == nil {
		return "", "", "", errors.New("auth user store not initialized")
	}
	var model AuthUserModel
	if err := s.db.Where("username_norm = ?", usernameNorm).First(&model).Error; err != nil {
		return "", "", "", err
	}
	return model.Username, model.PasswordHash, model.PlayerID, nil
}
