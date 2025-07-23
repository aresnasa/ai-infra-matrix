package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	
	"gopkg.in/yaml.v3"
	"github.com/sirupsen/logrus"
)

type PlaybookService struct{}

func NewPlaybookService() *PlaybookService {
	return &PlaybookService{}
}

type PlaybookContent struct {
	Name    string                 `yaml:"name"`
	Hosts   string                 `yaml:"hosts"`
	Become  bool                   `yaml:"become"`
	Vars    map[string]interface{} `yaml:"vars,omitempty"`
	Tasks   []TaskContent          `yaml:"tasks"`
}

type TaskContent struct {
	Name   string                 `yaml:"name"`
	Module map[string]interface{} `yaml:",inline"`
}

func (s *PlaybookService) GeneratePlaybook(projectID uint, userID uint) (*models.PlaybookGeneration, error) {
	// 获取项目信息
	projectService := NewProjectService()
	rbacService := NewRBACService(database.DB)
	project, err := projectService.GetProject(projectID, userID, rbacService)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// 生成playbook内容
	playbookContent, err := s.buildPlaybookContent(project)
	if err != nil {
		return nil, fmt.Errorf("failed to build playbook content: %w", err)
	}

	// 生成文件名和路径
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("playbook-%s-%d.yml", 
		strings.ReplaceAll(project.Name, " ", "-"), 
		timestamp)
	
	// 使用绝对路径避免工作目录问题
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	outputDir := filepath.Join(currentDir, "outputs")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	
	filePath := filepath.Join(outputDir, fileName)

	// 写入文件
	if err := s.writePlaybookFile(filePath, playbookContent, project); err != nil {
		return nil, fmt.Errorf("failed to write playbook file: %w", err)
	}

	// 生成inventory文件
	inventoryPath := filepath.Join(outputDir, fmt.Sprintf("inventory-%s-%d.ini", 
		strings.ReplaceAll(project.Name, " ", "-"), 
		timestamp))
	
	if err := s.writeInventoryFile(inventoryPath, project.Hosts); err != nil {
		logrus.WithError(err).Warn("Failed to write inventory file")
	}

	// 记录生成历史
	generation := &models.PlaybookGeneration{
		ProjectID: projectID,
		FileName:  fileName,
		FilePath:  filePath,
		Status:    "success",
	}

	if err := database.DB.Create(generation).Error; err != nil {
		logrus.WithError(err).Error("Failed to save generation record")
	}

	logrus.WithFields(logrus.Fields{
		"project_id": projectID,
		"file_path":  filePath,
	}).Info("Playbook generated successfully")

	return generation, nil
}

func (s *PlaybookService) buildPlaybookContent(project *models.Project) ([]PlaybookContent, error) {
	// 构建变量映射
	vars := make(map[string]interface{})
	for _, variable := range project.Variables {
		vars[variable.Name] = s.parseVariableValue(variable.Value, variable.Type)
	}

	// 构建任务列表
	var tasks []TaskContent
	for _, task := range project.Tasks {
		if !task.Enabled {
			continue
		}

		taskContent := TaskContent{
			Name:   task.Name,
			Module: make(map[string]interface{}),
		}

		// 解析任务参数
		args := s.parseTaskArgs(task.Args)
		taskContent.Module[task.Module] = args

		tasks = append(tasks, taskContent)
	}

	// 如果没有任务，添加默认ping任务
	if len(tasks) == 0 {
		tasks = append(tasks, TaskContent{
			Name: "Test connection",
			Module: map[string]interface{}{
				"ping": nil,
			},
		})
	}

	playbook := []PlaybookContent{
		{
			Name:   project.Name,
			Hosts:  "all",
			Become: true,
			Vars:   vars,
			Tasks:  tasks,
		},
	}

	return playbook, nil
}

func (s *PlaybookService) parseVariableValue(value, varType string) interface{} {
	switch varType {
	case "number":
		// 尝试解析为数字
		if strings.Contains(value, ".") {
			if f, err := parseFloat(value); err == nil {
				return f
			}
		} else {
			if i, err := parseInt(value); err == nil {
				return i
			}
		}
		return value
	case "boolean":
		return strings.ToLower(value) == "true"
	case "list":
		// 简单的列表解析，用逗号分割
		items := strings.Split(value, ",")
		var result []string
		for _, item := range items {
			result = append(result, strings.TrimSpace(item))
		}
		return result
	default:
		return value
	}
}

func (s *PlaybookService) parseTaskArgs(args string) interface{} {
	if args == "" {
		return nil
	}

	// 如果参数包含=，解析为键值对
	if strings.Contains(args, "=") {
		result := make(map[string]interface{})
		pairs := strings.Split(args, " ")
		
		for _, pair := range pairs {
			if strings.Contains(pair, "=") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					result[key] = value
				}
			}
		}
		
		if len(result) > 0 {
			return result
		}
	}

	// 否则返回原始字符串
	return args
}

func (s *PlaybookService) writePlaybookFile(filePath string, content []PlaybookContent, project *models.Project) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 写入文件头注释
	header := fmt.Sprintf(`---
# Ansible Playbook - %s
# 生成时间: %s
# 描述: %s

`, project.Name, time.Now().Format("2006-01-02 15:04:05"), project.Description)

	if _, err := file.WriteString(header); err != nil {
		return err
	}

	// 写入YAML内容
	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	
	encoder.SetIndent(2)
	return encoder.Encode(content)
}

func (s *PlaybookService) writeInventoryFile(filePath string, hosts []models.Host) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 按组分类主机
	groups := make(map[string][]models.Host)
	for _, host := range hosts {
		group := host.Group
		if group == "" {
			group = "ungrouped"
		}
		groups[group] = append(groups[group], host)
	}

	// 写入inventory文件
	for group, groupHosts := range groups {
		if _, err := file.WriteString(fmt.Sprintf("[%s]\n", group)); err != nil {
			return err
		}

		for _, host := range groupHosts {
			line := fmt.Sprintf("%s ansible_host=%s ansible_port=%d ansible_user=%s\n",
				host.Name, host.IP, host.Port, host.User)
			if _, err := file.WriteString(line); err != nil {
				return err
			}
		}

		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}

	return nil
}

// GetPlaybookFile 获取生成的playbook文件路径和文件名
func (s *PlaybookService) GetPlaybookFile(generationID uint) (string, string, error) {
	var generation models.PlaybookGeneration
	if err := database.DB.First(&generation, generationID).Error; err != nil {
		logrus.WithFields(logrus.Fields{
			"generation_id": generationID,
			"error": err,
		}).Error("Generation record not found in database")
		return "", "", fmt.Errorf("generation record not found: %w", err)
	}

	// 确保使用绝对路径
	var filePath string
	if filepath.IsAbs(generation.FilePath) {
		filePath = generation.FilePath
	} else {
		// 如果数据库中是相对路径，转换为绝对路径
		currentDir, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("failed to get current directory: %w", err)
		}
		filePath = filepath.Join(currentDir, generation.FilePath)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logrus.WithFields(logrus.Fields{
			"generation_id": generationID,
			"file_path": filePath,
		}).Error("Playbook file not found on filesystem")
		return "", "", fmt.Errorf("playbook file not found: %s", filePath)
	}

	logrus.WithFields(logrus.Fields{
		"generation_id": generationID,
		"file_path": filePath,
		"file_name": generation.FileName,
	}).Info("Playbook file found successfully")

	return filePath, generation.FileName, nil
}

// 辅助函数
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func parseInt(s string) (int, error) {
	i, err := strconv.Atoi(s)
	return i, err
}
