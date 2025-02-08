package main

import (
	"time"
	"log"

	"cadence/internal/data"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type ServerConfig struct {
	Port            string
	ShutdownTimeout time.Duration
	Environment     string
	AllowedOrigins  []string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
}

type Server struct {
	config  *ServerConfig
	router  *gin.Engine
	repo    *data.Repository
	logger  *log.Logger
	metrics *Metrics
}

type Metrics struct {
	RequestCount  int64
	ErrorCount    int64
	LastErrorTime time.Time
}

func LoadServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:            GetEnvWithDefault("PORT", ":8080"),
		ShutdownTimeout: time.Duration(GetEnvAsIntWithDefault("SHUTDOWN_TIMEOUT_SECONDS", 5))*time.Second,
		Environment:     GetEnvWithDefault("ENVIRONMENT", "development"),
		AllowedOrigins:  GetEnvAsSlice("ALLOWED_ORIGINS", []string{"*"}),
		ReadTimeout:     time.Duration(GetEnvAsIntWithDefault("READ_TIMEOUT_SECONDS", 5))*time.Second,
		WriteTimeout:    time.Duration(GetEnvAsIntWithDefault("WRITE_TIMEOUT_SECONDS", 10))*time.Second,
		IdleTimeout:     time.Duration(GetEnvAsIntWithDefault("IDLE_TIMEOUT_SECONDS", 120))*time.Second,
	}
}

func NewServer(config *ServerConfig, repo *data.Repository, logger *log.Logger) *Server {
	if config.Environment != "development" {
		gin.SetMode(gin.ReleaseMode)
	}

	s := &Server{
		config:  config,
		router:  gin.New(),
		repo:    repo,
		logger:  logger,
		metrics: &Metrics{},
	}

	s.SetupMiddleware()
	s.SetupRoutes()

	return s
}