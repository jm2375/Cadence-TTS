package main

import (
	"net/http"
	"time"

	"cadence/internal/data"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) SetupMiddleware() {
	s.router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		s.logger.Printf("panic recovered: %v", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, APIError{
			Code:    http.StatusInternalServerError,
			Message: "Internal Server Error",
		})
	}))

	corsConfig := cors.Config{
		AllowOrigins:     s.config.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	s.router.Use(cors.New(corsConfig))

	s.router.Use(s.RequestIDMiddleware())
	s.router.Use(s.LoggingMiddleware())
	s.router.Use(s.MetricsMiddleware())
}

func (s *Server) RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = time.Now().Format("20060102150405") + RandString(6)
		}
		c.Set("requestID", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (s *Server) LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		s.logger.Printf("| %3d | %13v | %15s | %s | %s |",
			c.Writer.Status(),
			time.Since(start),
			c.ClientIP(),
			c.Request.Method,
			path,
		)
	}
}

func (s *Server) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.metrics.RequestCount++
		c.Next()
		if len(c.Errors) > 0 {
			s.metrics.ErrorCount++
			s.metrics.LastErrorTime = time.Now()
		}
	}
}

func (s *Server) AuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			s.SendError(c, http.StatusUnauthorized, "Missing authorization header", "")
			return
		}

		user, err := s.repo.ValidateToken(token)
		if err != nil {
			s.SendError(c, http.StatusUnauthorized, "Invalid token", err.Error())
			return
		}

		if err := s.repo.CheckAccess(user); err != nil {
			s.SendError(c, http.StatusTooManyRequests, "Rate limit exceeded", err.Error())
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func (s *Server) RequirePermission(_ string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			s.SendError(c, http.StatusUnauthorized, "User not found in context", "")
			return
		}

		if typedUser, ok := user.(*data.User); !ok || !typedUser.IsAdmin {
			s.SendError(c, http.StatusForbidden, "Insufficient permissions", "")
			return
		}

		c.Next()
	}
}

func (s *Server) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"metrics": s.metrics,
	})
}
