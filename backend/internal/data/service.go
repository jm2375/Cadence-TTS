package data

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (r *Repository) CreateUser(username string, password string, isAdmin bool) error {
	if err := r.ValidateUsername(username); err != nil {
		return errors.Join(ErrValidation, err)
	}
	
	if err := r.ValidatePassword(password); err != nil {
		return errors.Join(ErrValidation, err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), r.Config.BcryptCost)
	if err != nil {
		return errors.Join(ErrDatabase, err)
	}

	return r.DB.Transaction(func(tx *gorm.DB) error {
		user := &User{
			Username:     username,
			PasswordHash: hash,
			IsAdmin:      isAdmin,
			Usage: Usage{
				RateLimit:   r.Config.DefaultRateLimit,
				Capacity:    r.Config.DefaultCapacity,
				LastRequest: time.Now(),
			},
		}

		err := tx.Create(user).Error
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint") {
				return ErrUserExists
			}
			return errors.Join(ErrDatabase, err)
		}

		return nil
	})
}

func (r *Repository) AuthenticateUser(username string, password string) (*User, error) {
	var user User
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

func (r *Repository) CreateToken(user *User) (*Token, error) {
	tokenData := make([]byte, r.Config.TokenBytes)
	if _, err := rand.Read(tokenData); err != nil {
		return nil, errors.Join(ErrDatabase, err)
	}

	plainText := base64.URLEncoding.EncodeToString(tokenData)
	hash, err := bcrypt.GenerateFromPassword([]byte(plainText), r.Config.BcryptCost)
	if err != nil {
		return nil, errors.Join(ErrDatabase, err)
	}

	token := &Token{
		UserID:    user.ID,
		Hash:      hash,
		PlainText: plainText,
		ExpiresAt: time.Now().Add(time.Millisecond * time.Duration(r.Config.TokenExpiry)),
	}

	if err := r.DB.Create(token).Error; err != nil {
		return nil, errors.Join(ErrDatabase, err)
	}

	return token, nil
}

func (r *Repository) ValidateToken(tokenString string) (*User, error) {
	var tokens []Token
	if err := r.DB.Where("expires_at > ?", time.Now()).Find(&tokens).Error; err != nil {
		return nil, errors.Join(ErrDatabase, err)
	}

	for _, token := range tokens {
		if err := bcrypt.CompareHashAndPassword(token.Hash, []byte(tokenString)); err == nil {
			var user User
			if err := r.DB.First(&user, token.UserID).Error; err != nil {
				return nil, errors.Join(ErrDatabase, err)
			}
			return &user, nil
		}
	}

	return nil, ErrInvalidCredentials
}

func (r *Repository) CheckAccess(user *User) error {
	if user.IsAdmin {
		return nil
	}

	now := time.Now()
	window := now.Add(-time.Minute)

	if user.Usage.TotalRequests >= user.Usage.Capacity {
		return ErrCapacityExceeded
	}

	if user.Usage.LastRequest.Before(window) {
		user.Usage.RateCount = 1
	} else if user.Usage.RateCount >= user.Usage.RateLimit {
		return ErrRateLimitExceeded
	} else {
		user.Usage.RateCount++
	}

	user.Usage.LastRequest = now
	user.Usage.TotalRequests++

	if err := r.DB.Save(user).Error; err != nil {
		return errors.Join(ErrDatabase, err)
	}

	return nil
}

func (r *Repository) UpdateLimits(userID uint, rateLimit int, capacity int) error {
	result := r.DB.Model(&User{}).Where("id = ?", userID).
		Updates(Usage{RateLimit: rateLimit, Capacity: capacity})

	if result.Error != nil {
		return errors.Join(ErrDatabase, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrInvalidCredentials
	}

	return nil
}
