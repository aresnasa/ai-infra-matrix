# SaltStack集成环境变量配置修复报告

## 修复概述
修复了AI-Infra-Matrix中SaltStack集成的硬编码配置问题，将所有相关配置改为通过环境变量进行管理，提供了更好的配置灵活性和部署适配性。

## 修复的具体问题
1. **硬编码的SaltStack Master URL**: `http://saltstack:8000`
2. **硬编码的SaltStack Master Host**: `saltstack` 和 `salt-master`
3. **缺少API Token配置支持**
4. **环境变量配置不统一**

## 修改的文件清单

### 1. 后端服务配置

#### `src/backend/internal/services/saltstack_service.go`
- **修改内容**: 
  - 添加了`os`包导入
  - 修改`NewSaltStackService()`函数，从环境变量读取配置
  - 支持`SALTSTACK_MASTER_URL`和`SALTSTACK_API_TOKEN`环境变量
- **环境变量**:
  ```go
  masterURL := os.Getenv("SALTSTACK_MASTER_URL")
  if masterURL == "" {
      masterURL = "http://saltstack:8000" // 默认值
  }
  apiToken := os.Getenv("SALTSTACK_API_TOKEN")
  ```

#### `src/backend/internal/controllers/slurm_controller.go`
- **修改内容**:
  - 添加了`os`包导入
  - 添加`getSaltStackMasterHost()`辅助函数
  - 修改所有使用`SaltStackDeploymentConfig`的地方，使用环境变量
- **环境变量**:
  ```go
  func getSaltStackMasterHost() string {
      masterHost := os.Getenv("SALTSTACK_MASTER_HOST")
      if masterHost == "" {
          masterHost = "saltstack" // 默认容器名
      }
      return masterHost
  }
  ```
- **影响的函数**:
  - SSH部署Minion相关函数
  - 集群初始化函数
  - 单节点部署函数

### 2. 环境配置文件

#### `.env.example`
- **修改内容**: 更新SaltStack配置部分，添加了详细的环境变量说明
- **新增配置**:
  ```bash
  # SaltStack Master 主机地址 (容器名或IP地址)
  SALTSTACK_MASTER_HOST=saltstack
  
  # SaltStack Master API URL (包含协议和端口)
  SALTSTACK_MASTER_URL=http://saltstack:8000
  
  # SaltStack API 认证令牌 (可选，如果启用了API认证)
  SALTSTACK_API_TOKEN=
  
  # SaltStack Master 端口配置
  SALT_MASTER_HOST=saltstack
  SALT_MASTER_PORT=4505
  SALT_RETURN_PORT=4506
  SALT_API_PORT=8000
  ```

### 3. 构建脚本

#### `build.sh`
- **修改内容**:
  - 添加`setup_saltstack_defaults()`函数
  - 在环境文件创建过程中自动设置SaltStack默认配置
  - 确保环境变量的一致性和完整性
- **新增功能**:
  ```bash
  # 设置SaltStack默认配置
  setup_saltstack_defaults() {
      local env_file="$1"
      
      # SaltStack Master 主机配置
      if ! grep -q "^SALTSTACK_MASTER_HOST=" "$env_file" 2>/dev/null; then
          set_or_update_env_var "SALTSTACK_MASTER_HOST" "saltstack" "$env_file"
      fi
      
      # 其他配置...
  }
  ```

### 4. Docker Compose配置

#### `docker-compose.yml`
- **修改内容**: 在backend服务的environment部分添加SaltStack环境变量
- **新增环境变量**:
  ```yaml
  # SaltStack 配置
  SALTSTACK_MASTER_HOST: "${SALTSTACK_MASTER_HOST:-saltstack}"
  SALTSTACK_MASTER_URL: "${SALTSTACK_MASTER_URL:-http://saltstack:8000}"
  SALTSTACK_API_TOKEN: "${SALTSTACK_API_TOKEN:-}"
  ```

## 支持的环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SALTSTACK_MASTER_HOST` | `saltstack` | SaltStack Master容器名或IP地址 |
| `SALTSTACK_MASTER_URL` | `http://saltstack:8000` | SaltStack API完整URL |
| `SALTSTACK_API_TOKEN` | 空 | SaltStack API认证令牌（可选） |

## 配置使用方法

### 1. 开发环境配置
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置（如需修改默认值）
vim .env

# 应用配置并构建
./build.sh create-env dev
./build.sh build backend
```

### 2. 生产环境配置
```bash
# 创建生产环境配置
./build.sh create-env prod

# 编辑生产环境配置
vim .env.prod

# 使用生产配置构建
ENV_FILE=.env.prod ./build.sh build backend
```

### 3. 自定义SaltStack Master地址
```bash
# 设置自定义Master地址
export SALTSTACK_MASTER_HOST="custom-saltstack-host"
export SALTSTACK_MASTER_URL="http://custom-saltstack-host:8000"

# 或在.env文件中设置
echo "SALTSTACK_MASTER_HOST=custom-saltstack-host" >> .env
echo "SALTSTACK_MASTER_URL=http://custom-saltstack-host:8000" >> .env
```

## 验证修复效果

### 1. 检查环境变量配置
```bash
# 检查.env文件中的SaltStack配置
grep -E "^SALTSTACK|^SALT_" .env

# 检查Docker Compose中的变量传递
docker-compose config | grep -A 10 -B 5 SALTSTACK
```

### 2. 验证后端服务配置
```bash
# 构建并启动服务
./build.sh build backend
docker-compose up -d backend saltstack

# 检查后端日志中的SaltStack连接信息
docker-compose logs backend | grep -i salt
```

### 3. 测试Minion安装功能
通过前端界面或API测试SSH部署SaltStack Minion功能，确认：
- 正确读取环境变量中的Master地址
- Minion配置文件包含正确的Master信息
- 能够成功连接到SaltStack Master

## 兼容性说明
- **向后兼容**: 如果未设置环境变量，将使用默认值，保持现有行为
- **灵活配置**: 支持容器名、IP地址、域名等多种Master地址格式
- **可选认证**: API Token为可选配置，适配不同的SaltStack部署方式

## 注意事项
1. **环境变量优先级**: 环境变量 > .env文件 > 默认值
2. **容器网络**: 确保SaltStack容器在正确的Docker网络中
3. **端口配置**: 如果修改了SaltStack API端口，需要同时更新URL配置
4. **认证配置**: 如果启用了SaltStack API认证，需要设置有效的API Token

## 后续建议
1. **监控集成**: 添加SaltStack连接状态监控
2. **配置验证**: 在应用启动时验证SaltStack配置的有效性
3. **自动发现**: 支持自动发现SaltStack服务的网络地址
4. **配置热重载**: 支持运行时更新SaltStack配置而无需重启