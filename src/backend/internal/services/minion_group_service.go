package services

import (
	"errors"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/gorm"
)

// MinionGroupService 提供 Minion 分组管理功能
type MinionGroupService struct {
	db *gorm.DB
}

// NewMinionGroupService 创建新的 MinionGroupService 实例
func NewMinionGroupService() *MinionGroupService {
	return &MinionGroupService{
		db: database.DB,
	}
}

// CreateGroup 创建分组
func (s *MinionGroupService) CreateGroup(group *models.MinionGroup) error {
	return s.db.Create(group).Error
}

// UpdateGroup 更新分组
func (s *MinionGroupService) UpdateGroup(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.MinionGroup{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteGroup 删除分组
func (s *MinionGroupService) DeleteGroup(id uint) error {
	// 先删除成员关系
	if err := s.db.Where("group_id = ?", id).Delete(&models.MinionGroupMembership{}).Error; err != nil {
		return err
	}
	// 再删除分组
	return s.db.Delete(&models.MinionGroup{}, id).Error
}

// GetGroup 获取单个分组
func (s *MinionGroupService) GetGroup(id uint) (*models.MinionGroup, error) {
	var group models.MinionGroup
	err := s.db.First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetGroupByName 根据名称获取分组
func (s *MinionGroupService) GetGroupByName(name string) (*models.MinionGroup, error) {
	var group models.MinionGroup
	err := s.db.Where("name = ?", name).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// ListGroups 获取所有分组
func (s *MinionGroupService) ListGroups() ([]models.MinionGroup, error) {
	var groups []models.MinionGroup
	err := s.db.Order("priority DESC, name ASC").Find(&groups).Error
	return groups, err
}

// AddMinionToGroup 将 Minion 添加到分组
func (s *MinionGroupService) AddMinionToGroup(minionID string, groupID uint) error {
	// 检查是否已存在
	var existing models.MinionGroupMembership
	err := s.db.Where("minion_id = ? AND group_id = ?", minionID, groupID).First(&existing).Error
	if err == nil {
		// 已存在，不需要重复添加
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	membership := models.MinionGroupMembership{
		MinionID: minionID,
		GroupID:  groupID,
	}
	return s.db.Create(&membership).Error
}

// RemoveMinionFromGroup 从分组移除 Minion
func (s *MinionGroupService) RemoveMinionFromGroup(minionID string, groupID uint) error {
	return s.db.Where("minion_id = ? AND group_id = ?", minionID, groupID).Delete(&models.MinionGroupMembership{}).Error
}

// SetMinionGroup 设置 Minion 的分组（替换所有现有分组）
func (s *MinionGroupService) SetMinionGroup(minionID string, groupName string) error {
	// 先删除该 Minion 的所有分组关系
	if err := s.db.Where("minion_id = ?", minionID).Delete(&models.MinionGroupMembership{}).Error; err != nil {
		return err
	}

	if groupName == "" {
		return nil
	}

	// 查找或创建分组
	var group models.MinionGroup
	err := s.db.Where("name = ?", groupName).First(&group).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 创建新分组
		group = models.MinionGroup{
			Name:        groupName,
			DisplayName: groupName,
		}
		if err := s.db.Create(&group).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// 添加成员关系
	membership := models.MinionGroupMembership{
		MinionID: minionID,
		GroupID:  group.ID,
	}
	return s.db.Create(&membership).Error
}

// GetMinionGroups 获取 Minion 所属的分组
func (s *MinionGroupService) GetMinionGroups(minionID string) ([]models.MinionGroup, error) {
	var groups []models.MinionGroup
	err := s.db.Joins("JOIN minion_group_memberships ON minion_group_memberships.group_id = minion_groups.id").
		Where("minion_group_memberships.minion_id = ?", minionID).
		Find(&groups).Error
	return groups, err
}

// GetMinionGroupName 获取 Minion 的主分组名称（第一个分组）
func (s *MinionGroupService) GetMinionGroupName(minionID string) string {
	var membership models.MinionGroupMembership
	err := s.db.Preload("Group").Where("minion_id = ?", minionID).First(&membership).Error
	if err != nil {
		return ""
	}
	return membership.Group.Name
}

// GetGroupMinions 获取分组内的所有 Minion ID
func (s *MinionGroupService) GetGroupMinions(groupID uint) ([]string, error) {
	var memberships []models.MinionGroupMembership
	err := s.db.Where("group_id = ?", groupID).Find(&memberships).Error
	if err != nil {
		return nil, err
	}

	minionIDs := make([]string, len(memberships))
	for i, m := range memberships {
		minionIDs[i] = m.MinionID
	}
	return minionIDs, nil
}

// GetAllMinionGroupMap 获取所有 Minion 的分组映射
func (s *MinionGroupService) GetAllMinionGroupMap() (map[string]string, error) {
	var memberships []models.MinionGroupMembership
	err := s.db.Preload("Group").Find(&memberships).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, m := range memberships {
		if m.Group.Name != "" {
			result[m.MinionID] = m.Group.Name
		}
	}
	return result, nil
}

// BatchSetMinionGroups 批量设置 Minion 分组
func (s *MinionGroupService) BatchSetMinionGroups(minionGroups map[string]string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for minionID, groupName := range minionGroups {
			// 先删除该 Minion 的所有分组关系
			if err := tx.Where("minion_id = ?", minionID).Delete(&models.MinionGroupMembership{}).Error; err != nil {
				return err
			}

			if groupName == "" {
				continue
			}

			// 查找或创建分组
			var group models.MinionGroup
			err := tx.Where("name = ?", groupName).First(&group).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 创建新分组
				group = models.MinionGroup{
					Name:        groupName,
					DisplayName: groupName,
				}
				if err := tx.Create(&group).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}

			// 添加成员关系
			membership := models.MinionGroupMembership{
				MinionID: minionID,
				GroupID:  group.ID,
			}
			if err := tx.Create(&membership).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
