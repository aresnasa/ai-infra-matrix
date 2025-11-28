# 脚本模板管理指南

## 概述

本目录包含 Salt Minion 安装/卸载和系统检测的脚本模板文件。这些模板使用 Go 的 `text/template` 语法，支持动态参数替换。

## 目录结构

```
scripts/templates/
├── README.md                          # 本文档
├── salt-install-debian.sh.tmpl        # Debian/Ubuntu Salt 安装模板
├── salt-install-rhel.sh.tmpl          # RHEL/CentOS/Rocky 等 Salt 安装模板  
├── salt-install-generic.sh.tmpl       # 通用 Salt 安装模板（使用 Bootstrap）
├── salt-uninstall-debian.sh.tmpl      # Debian/Ubuntu Salt 卸载模板
├── salt-uninstall-rhel.sh.tmpl        # RHEL/CentOS/Rocky 等 Salt 卸载模板
├── salt-uninstall-generic.sh.tmpl     # 通用 Salt 卸载模板
├── os-detect.sh.tmpl                  # 操作系统检测脚本
└── ssh-test.sh.tmpl                   # SSH 连接测试脚本
```

## 模板语法

模板使用 Go `text/template` 语法，变量使用 `{{.VariableName}}` 格式：

### Salt 安装参数 (SaltInstallParams)

| 变量 | 类型 | 说明 |
|------|------|------|
| `{{.AppHubURL}}` | string | AppHub 服务器地址，用于下载安装包 |
| `{{.MasterHost}}` | string | Salt Master 主机地址 |
| `{{.MinionID}}` | string | Minion 标识符 |
| `{{.Version}}` | string | Salt 版本号 (如 3006.4) |
| `{{.Arch}}` | string | DEB 包架构 (amd64, arm64) |
| `{{.RpmArch}}` | string | RPM 包架构 (x86_64, aarch64) |
| `{{.SudoPrefix}}` | string | sudo 前缀 (空字符串或 "sudo ") |
| `{{.OS}}` | string | 操作系统类型 (ubuntu, debian, centos, rhel, etc.) |
| `{{.OSVersion}}` | string | 操作系统版本 |

### Salt 卸载参数 (SaltUninstallParams)

| 变量 | 类型 | 说明 |
|------|------|------|
| `{{.SudoPrefix}}` | string | sudo 前缀 |
| `{{.OS}}` | string | 操作系统类型 |

### SSH 测试参数 (SSHTestParams)

| 变量 | 类型 | 说明 |
|------|------|------|
| `{{.SudoPass}}` | string | sudo 密码（用于测试 sudo 权限）|

## 运维指南

### 修改模板

1. **外部模板优先**：系统优先从 `SCRIPTS_DIR` 环境变量指定的目录加载模板
2. **默认路径**：如未设置环境变量，默认从可执行文件同目录的 `scripts` 文件夹加载
3. **回退机制**：如果外部模板不存在，使用程序内嵌的模板

### 部署外部模板

```bash
# 设置外部脚本目录
export SCRIPTS_DIR=/opt/ai-infra-matrix/scripts

# 复制模板文件到外部目录
mkdir -p $SCRIPTS_DIR/templates
cp -r scripts/templates/* $SCRIPTS_DIR/templates/
```

### 热更新模板

修改模板后，无需重启程序，可通过 API 清除缓存：

```bash
# 清除所有模板缓存
curl -X POST http://localhost:8000/api/scripts/reload

# 或调用 ScriptLoader.ClearCache() / ReloadScript()
```

### 新增模板

1. 在 `scripts/templates/` 目录创建新的 `.tmpl` 文件
2. 在 `script_loader.go` 中的 `templateFiles` 映射表添加对应条目
3. 在对应的生成函数中添加调用逻辑

## 模板示例

### 条件判断

```bash
{{if .SudoPass}}
echo "使用 sudo 密码"
{{else}}
echo "无 sudo 密码"
{{end}}
```

### 变量使用

```bash
echo "Master: {{.MasterHost}}"
echo "Minion ID: {{.MinionID}}"
{{.SudoPrefix}}systemctl restart salt-minion
```

## 调试建议

1. **检查加载来源**：启用 debug 日志可看到脚本从哪里加载
2. **验证模板语法**：使用 `go template` 命令行工具验证
3. **查看生成脚本**：在日志中输出完整的生成脚本内容

## 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| 1.0 | 2024-01 | 初始版本，硬编码脚本 |
| 2.0 | 2024-01 | 重构为外部模板文件，支持运维热更新 |

## 注意事项

1. 模板文件必须是有效的 shell 脚本
2. 变量名区分大小写
3. 特殊字符需要正确转义
4. 建议在测试环境验证后再部署到生产
