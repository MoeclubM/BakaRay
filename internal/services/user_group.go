package services

import (
	"errors"

	"bakaray/internal/models"

	"gorm.io/gorm"
)

var ErrUserGroupNotFound = errors.New("用户组不存在")

// UserGroupService 用户组服务
type UserGroupService struct {
	db *gorm.DB
}

// NewUserGroupService 创建用户组服务
func NewUserGroupService(db *gorm.DB) *UserGroupService {
	return &UserGroupService{db: db}
}

// CreateUserGroup 创建用户组
func (s *UserGroupService) CreateUserGroup(name, description string) (*models.UserGroup, error) {
	group := &models.UserGroup{
		Name:        name,
		Description: description,
	}
	if err := s.db.Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// GetUserGroupByID 根据ID获取用户组
func (s *UserGroupService) GetUserGroupByID(id uint) (*models.UserGroup, error) {
	var group models.UserGroup
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserGroupNotFound
		}
		return nil, err
	}
	return &group, nil
}

// ListUserGroups 获取用户组列表
func (s *UserGroupService) ListUserGroups() ([]models.UserGroup, error) {
	var groups []models.UserGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// UpdateUserGroup 更新用户组
func (s *UserGroupService) UpdateUserGroup(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.UserGroup{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteUserGroup 删除用户组
func (s *UserGroupService) DeleteUserGroup(id uint) error {
	return s.db.Delete(&models.UserGroup{}, id).Error
}
