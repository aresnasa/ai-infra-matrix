package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/database"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/models"
	"gorm.io/gorm"
)

// InvitationCodeService 邀请码服务
type InvitationCodeService struct {
	db *gorm.DB
}

// NewInvitationCodeService 创建邀请码服务实例
func NewInvitationCodeService() *InvitationCodeService {
	return &InvitationCodeService{
		db: database.DB,
	}
}

// GenerateCode 生成随机邀请码
func (s *InvitationCodeService) GenerateCode() (string, error) {
	bytes := make([]byte, 8) // 16字符的十六进制字符串
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// 格式: XXXX-XXXX-XXXX-XXXX
	code := strings.ToUpper(hex.EncodeToString(bytes))
	return code[:4] + "-" + code[4:8] + "-" + code[8:12] + "-" + code[12:], nil
}

// CreateInvitationCode 创建邀请码
func (s *InvitationCodeService) CreateInvitationCode(req *models.CreateInvitationCodeRequest, createdBy uint) (*models.InvitationCode, error) {
	code, err := s.GenerateCode()
	if err != nil {
		return nil, errors.New("生成邀请码失败")
	}

	// 确保邀请码唯一
	for {
		var existing models.InvitationCode
		if err := s.db.Where("code = ?", code).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break // 邀请码不存在，可以使用
			}
			return nil, err
		}
		// 重新生成
		code, err = s.GenerateCode()
		if err != nil {
			return nil, errors.New("生成邀请码失败")
		}
	}

	maxUses := req.MaxUses
	if maxUses == 0 {
		maxUses = 1 // 默认只能使用一次
	}

	invitation := &models.InvitationCode{
		Code:         code,
		Description:  req.Description,
		RoleTemplate: req.RoleTemplate,
		MaxUses:      maxUses,
		UsedCount:    0,
		IsActive:     true,
		ExpiresAt:    req.ExpiresAt,
		CreatedBy:    createdBy,
	}

	if err := s.db.Create(invitation).Error; err != nil {
		return nil, errors.New("创建邀请码失败")
	}

	return invitation, nil
}

// BatchCreateInvitationCodes 批量创建邀请码
func (s *InvitationCodeService) BatchCreateInvitationCodes(req *models.CreateInvitationCodeRequest, createdBy uint) ([]*models.InvitationCode, error) {
	count := req.Count
	if count <= 0 {
		count = 1
	}
	if count > 100 {
		count = 100
	}

	var codes []*models.InvitationCode
	for i := 0; i < count; i++ {
		invitation, err := s.CreateInvitationCode(req, createdBy)
		if err != nil {
			return codes, err // 返回已创建的部分
		}
		codes = append(codes, invitation)
	}

	return codes, nil
}

// ValidateCode 验证邀请码是否有效
func (s *InvitationCodeService) ValidateCode(code string) (*models.InvitationCode, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, errors.New("邀请码不能为空")
	}

	var invitation models.InvitationCode
	if err := s.db.Where("code = ?", code).First(&invitation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("邀请码不存在")
		}
		return nil, err
	}

	if !invitation.IsValid() {
		if !invitation.IsActive {
			return nil, errors.New("邀请码已被禁用")
		}
		if invitation.ExpiresAt != nil && time.Now().After(*invitation.ExpiresAt) {
			return nil, errors.New("邀请码已过期")
		}
		if invitation.MaxUses > 0 && invitation.UsedCount >= invitation.MaxUses {
			return nil, errors.New("邀请码已达到使用次数上限")
		}
		return nil, errors.New("邀请码无效")
	}

	return &invitation, nil
}

// UseCode 使用邀请码
func (s *InvitationCodeService) UseCode(code string, userID uint, ipAddress string) error {
	code = strings.ToUpper(strings.TrimSpace(code))

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 锁定邀请码记录
	var invitation models.InvitationCode
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("code = ?", code).First(&invitation).Error; err != nil {
		tx.Rollback()
		return errors.New("邀请码不存在")
	}

	if !invitation.IsValid() {
		tx.Rollback()
		return errors.New("邀请码无效或已过期")
	}

	// 增加使用次数
	invitation.UsedCount++
	if err := tx.Save(&invitation).Error; err != nil {
		tx.Rollback()
		return errors.New("更新邀请码使用次数失败")
	}

	// 记录使用历史
	usage := &models.InvitationCodeUsage{
		InvitationCodeID: invitation.ID,
		UserID:           userID,
		UsedAt:           time.Now(),
		IPAddress:        ipAddress,
	}
	if err := tx.Create(usage).Error; err != nil {
		tx.Rollback()
		return errors.New("记录邀请码使用历史失败")
	}

	return tx.Commit().Error
}

// ListInvitationCodes 获取邀请码列表
func (s *InvitationCodeService) ListInvitationCodes(page, pageSize int, includeExpired bool) ([]models.InvitationCode, int64, error) {
	var codes []models.InvitationCode
	var total int64

	query := s.db.Model(&models.InvitationCode{})

	if !includeExpired {
		query = query.Where("is_active = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now())
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("Creator").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&codes).Error; err != nil {
		return nil, 0, err
	}

	return codes, total, nil
}

// GetInvitationCode 获取单个邀请码详情
func (s *InvitationCodeService) GetInvitationCode(id uint) (*models.InvitationCode, error) {
	var invitation models.InvitationCode
	if err := s.db.Preload("Creator").First(&invitation, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("邀请码不存在")
		}
		return nil, err
	}
	return &invitation, nil
}

// GetInvitationCodeUsages 获取邀请码使用记录
func (s *InvitationCodeService) GetInvitationCodeUsages(invitationCodeID uint) ([]models.InvitationCodeUsage, error) {
	var usages []models.InvitationCodeUsage
	if err := s.db.Where("invitation_code_id = ?", invitationCodeID).Preload("User").Order("used_at DESC").Find(&usages).Error; err != nil {
		return nil, err
	}
	return usages, nil
}

// DisableInvitationCode 禁用邀请码
func (s *InvitationCodeService) DisableInvitationCode(id uint) error {
	result := s.db.Model(&models.InvitationCode{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("邀请码不存在")
	}
	return nil
}

// EnableInvitationCode 启用邀请码
func (s *InvitationCodeService) EnableInvitationCode(id uint) error {
	result := s.db.Model(&models.InvitationCode{}).Where("id = ?", id).Update("is_active", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("邀请码不存在")
	}
	return nil
}

// DeleteInvitationCode 删除邀请码
func (s *InvitationCodeService) DeleteInvitationCode(id uint) error {
	// 先删除使用记录
	if err := s.db.Where("invitation_code_id = ?", id).Delete(&models.InvitationCodeUsage{}).Error; err != nil {
		return err
	}
	// 再删除邀请码
	result := s.db.Delete(&models.InvitationCode{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("邀请码不存在")
	}
	return nil
}

// GetStatistics 获取邀请码统计信息
func (s *InvitationCodeService) GetStatistics() (map[string]interface{}, error) {
	var totalCodes int64
	var activeCodes int64
	var totalUsages int64
	var expiredCodes int64

	s.db.Model(&models.InvitationCode{}).Count(&totalCodes)
	s.db.Model(&models.InvitationCode{}).Where("is_active = ? AND (expires_at IS NULL OR expires_at > ?)", true, time.Now()).Count(&activeCodes)
	s.db.Model(&models.InvitationCodeUsage{}).Count(&totalUsages)
	// 统计已过期的邀请码（设置了过期时间且已过期的）
	s.db.Model(&models.InvitationCode{}).Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).Count(&expiredCodes)

	return map[string]interface{}{
		"total_codes":   totalCodes,
		"active_codes":  activeCodes,
		"total_usages":  totalUsages,
		"expired_codes": expiredCodes,
	}, nil
}
