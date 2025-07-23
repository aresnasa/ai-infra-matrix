package services

import (
	"fmt"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/cache"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
)

type VariableService struct{}

func NewVariableService() *VariableService {
	return &VariableService{}
}

func (s *VariableService) CreateVariable(variable *models.Variable) error {
	if err := database.DB.Create(variable).Error; err != nil {
		return fmt.Errorf("failed to create variable: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.VariablesKey(variable.ProjectID))
	cache.Delete(cache.ProjectKey(variable.ProjectID))
	
	return nil
}

func (s *VariableService) GetVariables(projectID uint) ([]models.Variable, error) {
	// 先从缓存获取
	var variables []models.Variable
	cacheKey := cache.VariablesKey(projectID)
	
	if err := cache.Get(cacheKey, &variables); err == nil {
		return variables, nil
	}

	// 缓存未命中，从数据库获取
	if err := database.DB.Where("project_id = ?", projectID).Find(&variables).Error; err != nil {
		return nil, fmt.Errorf("failed to get variables: %w", err)
	}

	// 存入缓存
	cache.Set(cacheKey, variables, 30*time.Minute)
	
	return variables, nil
}

func (s *VariableService) UpdateVariable(id uint, variable *models.Variable) error {
	variable.ID = id
	if err := database.DB.Save(variable).Error; err != nil {
		return fmt.Errorf("failed to update variable: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.VariablesKey(variable.ProjectID))
	cache.Delete(cache.ProjectKey(variable.ProjectID))
	
	return nil
}

func (s *VariableService) DeleteVariable(id uint) error {
	var variable models.Variable
	if err := database.DB.First(&variable, id).Error; err != nil {
		return fmt.Errorf("variable not found: %w", err)
	}

	if err := database.DB.Delete(&variable).Error; err != nil {
		return fmt.Errorf("failed to delete variable: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.VariablesKey(variable.ProjectID))
	cache.Delete(cache.ProjectKey(variable.ProjectID))
	
	return nil
}

type TaskService struct{}

func NewTaskService() *TaskService {
	return &TaskService{}
}

func (s *TaskService) CreateTask(task *models.Task) error {
	if err := database.DB.Create(task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.TasksKey(task.ProjectID))
	cache.Delete(cache.ProjectKey(task.ProjectID))
	
	return nil
}

func (s *TaskService) GetTasks(projectID uint) ([]models.Task, error) {
	// 先从缓存获取
	var tasks []models.Task
	cacheKey := cache.TasksKey(projectID)
	
	if err := cache.Get(cacheKey, &tasks); err == nil {
		return tasks, nil
	}

	// 缓存未命中，从数据库获取
	if err := database.DB.Where("project_id = ?", projectID).Order("order_num ASC").Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// 存入缓存
	cache.Set(cacheKey, tasks, 30*time.Minute)
	
	return tasks, nil
}

func (s *TaskService) UpdateTask(id uint, task *models.Task) error {
	task.ID = id
	if err := database.DB.Save(task).Error; err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.TasksKey(task.ProjectID))
	cache.Delete(cache.ProjectKey(task.ProjectID))
	
	return nil
}

func (s *TaskService) DeleteTask(id uint) error {
	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if err := database.DB.Delete(&task).Error; err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// 清除相关缓存
	cache.Delete(cache.TasksKey(task.ProjectID))
	cache.Delete(cache.ProjectKey(task.ProjectID))
	
	return nil
}
