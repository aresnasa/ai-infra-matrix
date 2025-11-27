# 敏感数据加密功能文档

## 概述

ai-infra-matrix v0.3.8+ 引入了数据库敏感数据加密功能，保护存储在数据库中的以下敏感信息：

- SSH 密码和用户名（SLURM 节点配置）
- SSH 私钥路径（SLURM 节点配置）
- AI 助手 API 密钥和密钥（AI 配置）

## 加密算法

- 使用 **AES-256-GCM** 对称加密算法
- 密钥通过 **SHA-256** 从环境变量 `ENCRYPTION_KEY` 派生
- 每次加密使用随机 **Nonce**，确保相同明文产生不同密文
- 加密数据前缀 `encrypted:` 用于识别已加密数据

## 配置

### 1. 设置加密密钥

在 `.env` 文件中配置加密密钥：

```bash
# 加密配置（用于敏感数据加密，如 SSH 密码、API 密钥等）
ENCRYPTION_KEY=your-encryption-key-change-in-production-32-bytes
```

**重要提示：**
- 生产环境必须使用强随机密钥
- 密钥长度建议至少 32 字符
- 密钥一旦设置，不要随意更改，否则将无法解密已加密的数据
- 建议使用以下命令生成安全密钥：
  ```bash
  openssl rand -base64 32
  ```

### 2. Docker Compose 配置

在 `docker-compose.yml.tpl` 中，backend 服务已自动配置：

```yaml
environment:
  - ENCRYPTION_KEY=${ENCRYPTION_KEY:-your-encryption-key-change-in-production-32-bytes}
```

## 工作原理

### 自动加密/解密

通过 GORM Hooks 实现透明加密：

1. **BeforeCreate/BeforeUpdate/BeforeSave**: 自动加密敏感字段后存储
2. **AfterFind**: 从数据库读取后自动解密

### 受保护的模型

#### SlurmNode（SLURM 节点）
| 字段 | 说明 |
|------|------|
| Username | SSH 用户名（加密存储） |
| Password | SSH 密码（加密存储） |
| KeyPath | SSH 私钥路径（不在 JSON 中暴露） |

#### SlurmCluster（SLURM 集群）
| 字段 | 说明 |
|------|------|
| MasterSSH.Username | Master 节点 SSH 用户名（加密存储） |
| MasterSSH.Password | Master 节点 SSH 密码（加密存储） |

#### AIAssistantConfig（AI 助手配置）
| 字段 | 说明 |
|------|------|
| APIKey | API 密钥（加密存储） |
| APISecret | API 密钥（加密存储） |

### API 响应安全

敏感字段在 JSON 序列化时被隐藏（`json:"-"`）：
- 前端无法直接获取密码/密钥的值
- 对于 AI 配置，提供 `has_api_key` 和 `has_api_secret` 布尔字段指示是否已配置

## 迁移现有数据

如果升级到 v0.3.8+，需要迁移现有的明文数据：

```bash
# 编译迁移工具
cd src/backend
go build -o migrate-encryption ./cmd/migrate-encryption/main.go

# 运行迁移
./migrate-encryption
```

迁移工具会：
1. 检查所有敏感字段
2. 识别未加密的明文数据
3. 自动加密并更新数据库
4. 输出迁移报告

## 安全最佳实践

1. **密钥管理**
   - 使用环境变量或密钥管理服务存储 `ENCRYPTION_KEY`
   - 不要在代码或配置文件中硬编码密钥
   - 定期轮换密钥（需要重新加密所有数据）

2. **备份**
   - 备份 `ENCRYPTION_KEY`，丢失将导致无法解密数据
   - 数据库备份与密钥应分开存储

3. **传输安全**
   - 始终使用 HTTPS 传输敏感数据
   - 内部服务间通信也应加密

4. **日志审计**
   - 敏感字段不会在日志中输出
   - 开启访问日志记录异常访问

## 故障排除

### 解密失败

如果遇到 "failed to decrypt" 错误：

1. 检查 `ENCRYPTION_KEY` 是否与加密时使用的密钥一致
2. 确认加密服务已正确初始化
3. 检查数据是否被损坏

### 明文数据未加密

如果发现数据库中仍有明文：

1. 确认 `ENCRYPTION_KEY` 已正确配置
2. 运行迁移工具
3. 检查 GORM Hooks 是否正常工作

## 技术细节

### 加密数据格式

```
encrypted:BASE64(nonce + ciphertext + tag)
```

- `nonce`: 12 字节随机数
- `ciphertext`: 加密后的数据
- `tag`: GCM 认证标签

### 向后兼容

- 未加密的数据可以被读取（`DecryptSafely` 返回原文）
- 系统自动检测 `encrypted:` 前缀识别加密数据
- 保存时自动加密未加密的数据

## 相关文件

- `src/backend/internal/utils/encryption.go` - 加密服务实现
- `src/backend/internal/utils/encryption_manager.go` - 全局加密管理器
- `src/backend/internal/models/slurm_cluster_models.go` - SLURM 模型加密 Hooks
- `src/backend/internal/models/ai_assistant.go` - AI 配置加密 Hooks
- `src/backend/cmd/migrate-encryption/main.go` - 数据迁移工具
