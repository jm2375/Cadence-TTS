package data

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials	= errors.New("invalid credentials")
	ErrUserExists			= errors.New("username already exists")
	ErrRateLimitExceeded	= errors.New("rate limit exceeded")
	ErrCapacityExceeded		= errors.New("capacity limit exceeded")
	ErrValidation			= errors.New("validation error")
	ErrDatabase				= errors.New("database error")
	ErrUnauthorized			= errors.New("unauthorized access")
)

type DataConfig struct {
	TokenExpiry      int
	DefaultRateLimit int
	DefaultCapacity  int
	BcryptCost       int
	TokenBytes       int
	MinUsernameLen   int
	MaxUsernameLen   int
	MinPasswordLen   int
	MaxPasswordLen   int
}

type Repository struct {
	DB				*gorm.DB
	Config			*DataConfig
}

type Base struct {
	ID				uint				`gorm:"primarykey" json:"id"`
	CreatedAt		time.Time			`gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt		time.Time			`gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt		gorm.DeletedAt		`gorm:"index" json:"-"`
}

type User struct {
	Base
	Username		string				`gorm:"size:255;not null;uniqueIndex" json:"username"`
	PasswordHash	[]byte				`gorm:"not null" json:"-"`
	IsAdmin			bool				`gorm:"default:false" json:"is_admin"`
	Usage			Usage				`gorm:"embedded" json:"usage"`
}

type Usage struct {
	RateLimit		int					`gorm:"not null" json:"rate_limit"`
	RateCount		int					`gorm:"default:0" json:"rate_count"`
	LastRequest		time.Time			`gorm:"index" json:"last_request"`
	Capacity		int					`gorm:"not null" json:"capacity"`
	TotalRequests	int					`gorm:"default:0" json:"total_requests"`
}

type Token struct {
	Base
	UserID			uint				`gorm:"not null;index" json:"-"`
	Hash			[]byte				`gorm:"not null" json:"-"`
	PlainText		string				`gorm:"-" json:"token,omitempty"`
	ExpiresAt		time.Time			`gorm:"not null;index" json:"expires_at"`
}

func LoadConfig() *DataConfig {
    return &DataConfig{
        TokenExpiry: 			GetEnvAsInt("TOKEN_EXPIRY"),
        DefaultRateLimit: 		GetEnvAsInt("DEFAULT_RATE_LIMIT"),
        DefaultCapacity: 		GetEnvAsInt("DEFAULT_CAPACITY"),
        BcryptCost: 			GetEnvAsInt("BCRYPT_COST"),
        TokenBytes: 			GetEnvAsInt("TOKEN_BYTES"),
        MinUsernameLen: 		GetEnvAsInt("MIN_USERNAME_LEN"),
        MaxUsernameLen: 		GetEnvAsInt("MAX_USERNAME_LEN"),
        MinPasswordLen: 		GetEnvAsInt("MIN_PASSWORD_LEN"),
        MaxPasswordLen: 		GetEnvAsInt("MAX_PASSWORD_LEN"),
    }
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		DB:     db,
		Config: LoadConfig(),
	}
}

func (r *Repository) AutoMigrate() error {
	if err := r.DB.AutoMigrate(&User{}, &Token{}); err != nil {
		return errors.Join(ErrDatabase, err)
	}
	return nil
}