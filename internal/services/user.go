package services

import (
	"errors"

	"bakaray/internal/config"
	"bakaray/internal/models"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("用户不存在")
	ErrUserExists        = errors.New("用户已存在")
	ErrInvalidPassword   = errors.New("密码错误")
	ErrInsufficientBalance = errors.New("余额不足")
)

// UserService 用户服务
type UserService struct {
	db        *gorm.DB
	redis     *redis.Client
	jwtSecret string
	jwtExp    int
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, redis *redis.Client) *UserService {
	cfg, err := config.Load()
	if err != nil {
		// 如果加载失败，使用默认值
		cfg = &config.Config{
			JWT: config.JWTConfig{
				Secret:     "your-jwt-secret-key",
				Expiration: 86400,
			},
		}
	}

	return &UserService{
		db:        db,
		redis:     redis,
		jwtSecret: cfg.JWT.Secret,
		jwtExp:    cfg.JWT.Expiration,
	}
}

// GetJWTSecret 获取 JWT 密钥
func (s *UserService) GetJWTSecret() string {
	return s.jwtSecret
}

// GetJWTExpiration 获取 JWT 过期时间
func (s *UserService) GetJWTExpiration() int {
	return s.jwtExp
}

// CreateUser 创建用户
func (s *UserService) CreateUser(username, password string, groupID uint) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err == nil {
		return nil, ErrUserExists
	}

	// 简单密码哈希（实际使用 bcrypt）
	hash := hashPassword(password)

	user = models.User{
		Username:     username,
		PasswordHash: hash,
		Balance:      0,
		UserGroupID:  groupID,
		Role:         "user",
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// VerifyPassword 验证密码
func (s *UserService) VerifyPassword(user *models.User, password string) bool {
	return checkPassword(password, user.PasswordHash)
}

// UpdateBalance 更新用户余额
func (s *UserService) UpdateBalance(userID uint, amount int64) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("balance", gorm.Expr("balance + ?", amount)).Error
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(page, pageSize int) ([]models.User, int64) {
	var users []models.User
	var total int64

	s.db.Model(&models.User{}).Count(&total)
	offset := (page - 1) * pageSize
	s.db.Offset(offset).Limit(pageSize).Find(&users)

	return users, total
}

// CountUsers 统计用户总数
func (s *UserService) CountUsers() int64 {
	var total int64
	s.db.Model(&models.User{}).Count(&total)
	return total
}

// hashPassword 使用 bcrypt 哈希密码
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

// checkPassword 验证密码
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(id uint, updates map[string]interface{}) error {
	// Frontend compatibility: accept is_admin boolean and map to role.
	if raw, ok := updates["is_admin"]; ok {
		if isAdmin, ok := raw.(bool); ok {
			if isAdmin {
				updates["role"] = "admin"
			} else {
				updates["role"] = "user"
			}
		}
		delete(updates, "is_admin")
	}
	return s.db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id uint) error {
	return s.db.Delete(&models.User{}, id).Error
}

// ChangePassword 修改密码
func (s *UserService) ChangePassword(id uint, oldPassword, newPassword string) error {
	user, err := s.GetUserByID(id)
	if err != nil {
		return err
	}

	if !s.VerifyPassword(user, oldPassword) {
		return ErrInvalidPassword
	}

	hash := hashPassword(newPassword)
	return s.db.Model(&models.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}
