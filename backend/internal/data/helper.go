package data

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func GetEnvAsInt(key string) int {
	value := os.Getenv(key)
	intValue, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("required env var %s is missing or invalid", key))
	}
	return intValue
}

func (r *Repository) ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return errors.New("username cannot be empty")
	}
	if len(username) < r.Config.MinUsernameLen || len(username) > r.Config.MaxUsernameLen {
		return errors.New("invalid username length")
	}
	return nil
}

func (r *Repository) ValidatePassword(password string) error {
	if len(password) < r.Config.MinPasswordLen || len(password) > r.Config.MaxPasswordLen {
		return errors.New("invalid password length")
	}

	var hasUpper, hasLower, hasNumber bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return errors.New("password must contain at least one uppercase letter, lowercase letter, and number")
	}
	return nil
}
