package main

import (
	"strings"
	"strconv"
	"time"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvAsIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func GetEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func InitLogger(env string) *log.Logger {
	flags := log.Ldate | log.Ltime
	if env == "development" {
		flags |= log.Lshortfile
	}
	return log.New(os.Stdout, "", flags)
}

func (s *Server) SendError(c *gin.Context, code int, message string, details string) {
	c.JSON(code, APIError{
		Code:    code,
		Message: message,
		Details: details,
	})
}

func (s *Server) SendSuccess(c *gin.Context, code int, data interface{}) {
	c.JSON(code, gin.H{
		"success": true,
		"data":    data,
	})
}

func RandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}