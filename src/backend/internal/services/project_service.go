package services

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"

	"github.com/sirupsen/logrus"
)

type ProjectService struct{}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}

// GetUserProjectGroups 获取用户的项目组权限
func (s *ProjectService) GetUserProjectGroups(userID uint) ([]uint, error) {
	var userGroupIDs []uint
	err := database.DB.Model(&models.UserGroupMembership{}).
		Where("user_id = ?", userID).
		Pluck("user_group_id", &userGroupIDs).Error
	return userGroupIDs, err
}

// CanAccessProject 检查用户是否可以访问项目
func (s *ProjectService) CanAccessProject(projectID, userID uint, rbacService *RBACService) bool {
	// 检查是否为管理员
	if rbacService.IsAdmin(userID) {
		return true
	}

	// 检查是否为项目所有者
	var project models.Project
	if err := database.DB.First(&project, projectID).Error; err != nil {
		return false
	}

	if project.UserID == userID {
		return true
	}

	// 检查是否通过用户组有访问权限
	_, err := s.GetUserProjectGroups(userID)
	if err != nil {
		return false
	}

	// 这里可以扩展为检查项目是否分配给了用户的用户组
	// 暂时返回false，后续可以添加项目-用户组关联表
	return false
}

func (s *ProjectService) CreateProject(project *models.Project, userID uint) error {
	project.UserID = userID // 设置项目所有者
	if err := database.DB.Create(project).Error; err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	// 清除项目列表缓存
	cache.Delete(cache.ProjectListKey())

	logrus.WithField("project_id", project.ID).Info("Project created successfully")
	return nil
}

func (s *ProjectService) GetProject(id uint, userID uint, rbacService *RBACService) (*models.Project, error) {
	// 先从缓存获取
	var project models.Project
	cacheKey := cache.ProjectKey(id)

	if err := cache.Get(cacheKey, &project); err == nil {
		// 检查用户权限 - 管理员或项目所有者
		if rbacService.IsAdmin(userID) || project.UserID == userID {
			return &project, nil
		}
		return nil, fmt.Errorf("access denied")
	}

	// 缓存未命中，从数据库获取
	query := database.DB.Preload("Hosts").Preload("Variables").Preload("Tasks")

	// 如果不是管理员，只能访问自己的项目
	if !rbacService.IsAdmin(userID) {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Where("id = ?", id).First(&project).Error; err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	// 存入缓存
	cache.Set(cacheKey, project, 30*time.Minute)

	return &project, nil
}

func (s *ProjectService) GetProjects(userID uint) ([]models.Project, error) {
	// 先从缓存获取
	var projects []models.Project
	cacheKey := cache.ProjectListKey()

	if err := cache.Get(cacheKey, &projects); err == nil {
		// 过滤用户的项目
		var userProjects []models.Project
		for _, project := range projects {
			if project.UserID == userID {
				userProjects = append(userProjects, project)
			}
		}
		return userProjects, nil
	}

	// 缓存未命中，从数据库获取用户的项目
	if err := database.DB.Where("user_id = ?", userID).Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	// 存入缓存（这里存储的是用户特定的项目）
	cache.Set(cacheKey+fmt.Sprintf("_user_%d", userID), projects, 15*time.Minute)

	return projects, nil
}

// GetProjectsWithRBAC 根据用户权限获取项目列表
func (s *ProjectService) GetProjectsWithRBAC(userID uint, rbacService *RBACService) ([]models.Project, error) {
	var projects []models.Project

	// 检查用户是否有列出项目的权限
	hasPermission := rbacService.CheckPermission(userID, "projects", "list", "*", "")
	logrus.WithFields(logrus.Fields{
		"user_id":        userID,
		"has_permission": hasPermission,
		"resource":       "projects",
		"verb":           "list",
		"scope":          "*",
	}).Info("Checking projects list permission")

	if !hasPermission {
		return nil, fmt.Errorf("access denied: no permission to list projects")
	}

	// 如果是管理员，返回所有项目
	if rbacService.IsAdmin(userID) {
		if err := database.DB.Preload("User").Find(&projects).Error; err != nil {
			return nil, fmt.Errorf("failed to get all projects: %w", err)
		}
		return projects, nil
	}

	// 普通用户只能看到自己的项目和所属用户组的项目
	// 先获取用户自己的项目
	if err := database.DB.Where("user_id = ?", userID).Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":       userID,
		"project_count": len(projects),
	}).Info("Retrieved user projects")

	// TODO: 后续可以添加用户组项目的逻辑
	// 获取用户所属用户组的项目

	return projects, nil
}

func (s *ProjectService) UpdateProject(id uint, userID uint, project *models.Project) error {
	// 检查项目是否属于当前用户
	var existingProject models.Project
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&existingProject).Error; err != nil {
		return fmt.Errorf("project not found or access denied: %w", err)
	}

	project.ID = id
	project.UserID = userID // 确保用户ID不被修改
	if err := database.DB.Save(project).Error; err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.ProjectKey(id))
	cache.Delete(cache.ProjectListKey())
	cache.Delete(cache.ProjectListKey() + fmt.Sprintf("_user_%d", userID))

	logrus.WithField("project_id", id).Info("Project updated successfully")
	return nil
}

func (s *ProjectService) DeleteProject(id uint, userID uint) error {
	return s.SoftDeleteProject(id, userID)
}

// SoftDeleteProject 软删除项目（移至回收站）
func (s *ProjectService) SoftDeleteProject(id uint, userID uint) error {
	// 检查项目是否属于当前用户
	var project models.Project
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
		return fmt.Errorf("project not found or access denied: %w", err)
	}

	// 使用GORM的软删除 - 需要对已查询到的记录进行删除
	if err := database.DB.Delete(&project).Error; err != nil {
		return fmt.Errorf("failed to soft delete project: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.ProjectKey(id))
	cache.Delete(cache.ProjectListKey())
	cache.Delete(cache.ProjectListKey() + fmt.Sprintf("_user_%d", userID))
	cache.Delete(cache.HostsKey(id))
	cache.Delete(cache.VariablesKey(id))
	cache.Delete(cache.TasksKey(id))

	logrus.WithField("project_id", id).Info("Project moved to trash successfully")
	return nil
}

// GetDeletedProjects 获取回收站项目
func (s *ProjectService) GetDeletedProjects(userID uint, rbacService *RBACService) ([]models.Project, error) {
	var projects []models.Project
	query := database.DB.Unscoped().Where("deleted_at IS NOT NULL")

	// 如果不是管理员，只能看到自己的项目
	if !rbacService.IsAdmin(userID) {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("failed to get deleted projects: %w", err)
	}

	return projects, nil
}

// RestoreProject 从回收站恢复项目
func (s *ProjectService) RestoreProject(id uint, userID uint, rbacService *RBACService) error {
	var project models.Project
	query := database.DB.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", id)

	// 如果不是管理员，只能恢复自己的项目
	if !rbacService.IsAdmin(userID) {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&project).Error; err != nil {
		return fmt.Errorf("project not found in trash or access denied: %w", err)
	}

	// 恢复项目（清除删除时间）
	if err := database.DB.Unscoped().Model(&project).Update("deleted_at", nil).Error; err != nil {
		return fmt.Errorf("failed to restore project: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.ProjectKey(id))
	cache.Delete(cache.ProjectListKey())
	cache.Delete(cache.ProjectListKey() + fmt.Sprintf("_user_%d", userID))

	logrus.WithField("project_id", id).Info("Project restored from trash successfully")
	return nil
}

// ForceDeleteProject 永久删除项目（仅管理员）
func (s *ProjectService) ForceDeleteProject(id uint, userID uint) error {
	var project models.Project
	if err := database.DB.Unscoped().Where("id = ? AND deleted_at IS NOT NULL", id).First(&project).Error; err != nil {
		return fmt.Errorf("project not found in trash: %w", err)
	}

	// 永久删除项目及其关联数据
	if err := database.DB.Unscoped().Delete(&models.Project{}, id).Error; err != nil {
		return fmt.Errorf("failed to permanently delete project: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.ProjectKey(id))
	cache.Delete(cache.ProjectListKey())
	cache.Delete(cache.HostsKey(id))
	cache.Delete(cache.VariablesKey(id))
	cache.Delete(cache.TasksKey(id))

	logrus.WithField("project_id", id).Info("Project permanently deleted")
	return nil
}
