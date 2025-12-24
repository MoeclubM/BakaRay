package services

import (
	"errors"

	"bakaray/internal/models"

	"gorm.io/gorm"
)

var ErrNodeGroupNotFound = errors.New("节点组不存在")

// NodeGroupService 节点组服务
type NodeGroupService struct {
	db *gorm.DB
}

// NewNodeGroupService 创建节点组服务
func NewNodeGroupService(db *gorm.DB) *NodeGroupService {
	return &NodeGroupService{db: db}
}

// CreateNodeGroup 创建节点组
func (s *NodeGroupService) CreateNodeGroup(name, nodeType, description string) (*models.NodeGroup, error) {
	group := &models.NodeGroup{
		Name:        name,
		Type:        nodeType,
		Description: description,
	}
	if err := s.db.Create(group).Error; err != nil {
		return nil, err
	}
	return group, nil
}

// GetNodeGroupByID 根据ID获取节点组
func (s *NodeGroupService) GetNodeGroupByID(id uint) (*models.NodeGroup, error) {
	var group models.NodeGroup
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeGroupNotFound
		}
		return nil, err
	}
	return &group, nil
}

// ListNodeGroups 获取节点组列表
func (s *NodeGroupService) ListNodeGroups() ([]models.NodeGroup, error) {
	var groups []models.NodeGroup
	if err := s.db.Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

// UpdateNodeGroup 更新节点组
func (s *NodeGroupService) UpdateNodeGroup(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.NodeGroup{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteNodeGroup 删除节点组
func (s *NodeGroupService) DeleteNodeGroup(id uint) error {
	return s.db.Delete(&models.NodeGroup{}, id).Error
}
