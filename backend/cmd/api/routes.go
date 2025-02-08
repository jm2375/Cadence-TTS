package main

import (
	"net/http"
	"strconv"
	"errors"

	"cadence/internal/data"
	"cadence/internal/speech"

	"github.com/gin-gonic/gin"
)

func (s *Server) SetupRoutes() {
	s.router.GET("/health", s.HealthCheck)
	s.SetupAuthRoutes()

	api := s.router.Group("/api")
	api.Use(s.AuthenticationMiddleware())
	{
		s.SetupUserRoutes(api)
		s.SetupSpeechRoutes(api)
	}
}

func (s *Server) SetupAuthRoutes() {
	auth := s.router.Group("/auth")
	{
		auth.POST("/register", s.HandleRegister)
		auth.POST("/login", s.HandleLogin)
	}
}

func (s *Server) SetupUserRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("", s.RequirePermission("users:read"), s.HandleListUsers)
		users.POST("", s.RequirePermission("users:write"), s.HandleCreateUser)
		users.GET("/:id", s.RequirePermission("users:read"), s.HandleGetUser)
		users.PUT("/:id", s.RequirePermission("users:write"), s.HandleUpdateUser)
	}
}

func (s *Server) SetupSpeechRoutes(rg *gin.RouterGroup) {
	speech := rg.Group("/speech")
	{
		speech.GET("/voices", s.HandleListVoices)
		speech.POST("/synthesize", s.HandleSynthesize)
	}
}

func (s *Server) HandleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.SendError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := s.repo.CreateUser(req.Username, req.Password, false); err != nil {
		switch err {
		case data.ErrUserExists:
			s.SendError(c, http.StatusConflict, "Username already exists", "")
		case data.ErrValidation:
			s.SendError(c, http.StatusBadRequest, "Validation error", err.Error())
		default:
			s.logger.Printf("registration error: %v", err)
			s.SendError(c, http.StatusInternalServerError, "Failed to create user", "")
		}
		return
	}

	s.SendSuccess(c, http.StatusCreated, nil)
}

func (s *Server) HandleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.SendError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	user, err := s.repo.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		s.SendError(c, http.StatusUnauthorized, "Invalid credentials", "")
		return
	}

	token, err := s.repo.CreateToken(user)
	if err != nil {
		s.logger.Printf("token creation error: %v", err)
		s.SendError(c, http.StatusInternalServerError, "Failed to create token", "")
		return
	}

	s.SendSuccess(c, http.StatusOK, gin.H{
		"token":      token.PlainText,
		"expires_at": token.ExpiresAt,
	})
}

func (s *Server) HandleListUsers(c *gin.Context) {
	var users []data.User
	if err := s.repo.DB.Find(&users).Error; err != nil {
		s.logger.Printf("user list error: %v", err)
		s.SendError(c, http.StatusInternalServerError, "Failed to list users", "")
		return
	}

	s.SendSuccess(c, http.StatusOK, users)
}

func (s *Server) HandleCreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		IsAdmin  bool   `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.SendError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	if err := s.repo.CreateUser(req.Username, req.Password, req.IsAdmin); err != nil {
		switch err {
		case data.ErrUserExists:
			s.SendError(c, http.StatusConflict, "Username already exists", "")
		case data.ErrValidation:
			s.SendError(c, http.StatusBadRequest, "Validation error", err.Error())
		default:
			s.logger.Printf("user creation error: %v", err)
			s.SendError(c, http.StatusInternalServerError, "Failed to create user", "")
		}
		return
	}

	s.SendSuccess(c, http.StatusCreated, nil)
}

func (s *Server) HandleGetUser(c *gin.Context) {
	id := c.Param("id")
	var user data.User

	if err := s.repo.DB.First(&user, id).Error; err != nil {
		s.SendError(c, http.StatusNotFound, "User not found", "")
		return
	}

	s.SendSuccess(c, http.StatusOK, user)
}

func (s *Server) HandleUpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		RateLimit int `json:"rate_limit"`
		Capacity  int `json:"capacity"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.SendError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		s.SendError(c, http.StatusBadRequest, "Invalid user ID", "")
		return
	}

	if err := s.repo.UpdateLimits(uint(userID), req.RateLimit, req.Capacity); err != nil {
		s.logger.Printf("user update error: %v", err)
		s.SendError(c, http.StatusInternalServerError, "Failed to update user", "")
		return
	}

	s.SendSuccess(c, http.StatusOK, nil)
}

func (s *Server) HandleListVoices(c *gin.Context) {
	voices, err := speech.GetVoices()
	if err != nil {
		s.logger.Printf("voice list error: %v", err)
		s.SendError(c, http.StatusInternalServerError, "Failed to fetch voices", "")
		return
	}

	s.SendSuccess(c, http.StatusOK, voices)
}

func (s *Server) HandleSynthesize(c *gin.Context) {
	var req speech.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		s.SendError(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	audioData, err := req.Synthesize()
	if err != nil {
		var errMsg string
		var statusCode int

		switch {
		case errors.Is(err, speech.ErrInvalidInput):
			statusCode = http.StatusBadRequest
			errMsg = "Invalid synthesis request"
		case errors.Is(err, speech.ErrServiceFailure):
			statusCode = http.StatusServiceUnavailable
			errMsg = "Speech service unavailable"
		default:
			s.logger.Printf("synthesis error: %v", err)
			statusCode = http.StatusInternalServerError
			errMsg = "Failed to synthesize speech"
		}

		s.SendError(c, statusCode, errMsg, err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=speech.mp3")
	c.Data(http.StatusOK, "audio/mp3", audioData)
}