package database

import (
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/config"
	"github.com/aresnasa/ai-infra-matrix/src/backend/internal/utils"
)

var (
	// CryptoService 全局加密服务实例
	CryptoService *utils.CryptoService
)

// InitCrypto 初始化加密服务
func InitCrypto(cfg *config.Config) {
	CryptoService = utils.NewCryptoService(cfg.EncryptionKey)
}
