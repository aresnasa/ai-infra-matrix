# Salt Minion 部署脚本

此目录包含用于部署Salt Minion的标准化脚本。脚本按数字顺序执行。

## 脚本列表

1. **01-install-salt-minion.sh** - 安装salt-minion包
   - 环境变量:
     - `APPHUB_URL`: AppHub服务地址 (可选，如: http://apphub:80)
   - 安装优先级: AppHub仓库 → 系统仓库 → 官方仓库 → Bootstrap脚本

2. **02-configure-minion.sh** - 配置Minion连接Master
   - 环境变量:
     - `SALT_MASTER_HOST`: Master主机地址 (必需)
     - `SALT_MINION_ID`: Minion ID (可选，默认使用主机名)

3. **03-start-service.sh** - 启动salt-minion服务
   - 支持 systemd、SysV init 和直接启动

4. **04-verify-status.sh** - 验证服务状态
   - 检查服务状态、配置和日志

## 使用方法

### 手动执行

```bash
# 设置环境变量
export APPHUB_URL="http://apphub:80"
export SALT_MASTER_HOST="saltstack"
export SALT_MINION_ID="node01"

# 按顺序执行脚本
bash 01-install-salt-minion.sh
bash 02-configure-minion.sh
bash 03-start-service.sh
bash 04-verify-status.sh
```

### 自动批量执行

```bash
# 执行所有脚本
for script in *.sh; do
    echo "执行: $script"
    bash "$script" || exit 1
done
```

### 通过Go后端执行

Go服务会自动读取此目录中的脚本，按文件名排序后依次执行。

## 脚本规范

- 文件名格式: `NN-description.sh` (NN为两位数字)
- 所有脚本使用 `set -e` 遇错即停
- 使用环境变量传递参数
- 输出格式: `[Salt] ✓/✗/⚠ 消息`
- 成功返回 0，失败返回非0

## 扩展性

可以添加新脚本:
- `05-install-formulas.sh` - 安装Salt Formulas
- `06-apply-states.sh` - 应用初始状态
- `99-cleanup.sh` - 清理临时文件

只需确保文件名以数字开头即可自动被执行。
