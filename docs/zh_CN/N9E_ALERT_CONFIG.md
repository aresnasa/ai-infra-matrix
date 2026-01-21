# Nightingale 告警配置自动化工具

本工具用于自动化配置 Nightingale (N9E) 的监控和告警规则，支持全链路的监控配置管理。

## 功能特性

- ✅ 创建/管理业务组 (Busi Groups)
- ✅ 创建/管理告警规则 (Alert Rules)
- ✅ 创建/管理告警屏蔽规则 (Alert Mutes)
- ✅ 创建/管理告警订阅 (Alert Subscribes)
- ✅ 从 YAML 文件导入/导出告警规则
- ✅ 预定义告警规则模板
- ✅ 支持批量操作

## 快速开始

### 1. 安装依赖

```bash
pip3 install requests pyyaml python-dotenv
```

### 2. 配置环境变量

在项目根目录的 `.env` 文件中配置：

```bash
# Nightingale 配置
NIGHTINGALE_HOST=localhost
NIGHTINGALE_PORT=17000
N9E_USERNAME=root
N9E_PASSWORD=root.2020
```

### 3. 完整安装（推荐）

```bash
./scripts/n9e-setup.sh full-setup "AI Infrastructure"
```

这将自动完成：
1. 创建业务组
2. 导入预定义告警规则
3. 显示系统状态

## 使用方法

### Shell 脚本方式

```bash
# 查看帮助
./scripts/n9e-setup.sh help

# 初始化监控配置
./scripts/n9e-setup.sh init "My Business Group"

# 查看系统状态
./scripts/n9e-setup.sh status

# 列出业务组
./scripts/n9e-setup.sh list-groups

# 列出告警规则
./scripts/n9e-setup.sh list-rules 1

# 添加告警规则
./scripts/n9e-setup.sh add-rule "CPU告警" 'cpu_usage > 80' 2 1

# 添加预设告警规则
./scripts/n9e-setup.sh add-preset all 1       # 所有预设规则
./scripts/n9e-setup.sh add-preset cpu 1 90    # CPU 告警（阈值 90%）

# 导入告警规则
./scripts/n9e-setup.sh import config/n9e-alert-rules-example.yaml 1

# 导出告警规则
./scripts/n9e-setup.sh export my-rules.yaml 1
```

### Python 脚本方式

```bash
# 查看帮助
python3 scripts/n9e-alert-config.py --help

# 初始化监控配置
python3 scripts/n9e-alert-config.py init --group-name "My Business Group"

# 查看系统状态
python3 scripts/n9e-alert-config.py status

# 添加告警规则
python3 scripts/n9e-alert-config.py add-rule \
    --name "CPU使用率告警" \
    --prom-ql 'cpu_usage_active > 80' \
    --severity 2 \
    --group-id 1

# 导入告警规则
python3 scripts/n9e-alert-config.py import-rules \
    --file config/n9e-alert-rules-example.yaml \
    --group-id 1

# 导出告警规则
python3 scripts/n9e-alert-config.py export-rules \
    --group-id 1 \
    --output my-rules.yaml
```

## 告警规则 YAML 格式

创建告警规则配置文件 `my-rules.yaml`：

```yaml
rules:
  - name: "CPU使用率超过80%"
    note: "主机CPU使用率超过80%，请检查"
    prom_ql: "cpu_usage_active > 80"
    severity: 2  # 1:紧急 2:警告 3:通知
    interval: 15
    labels:
      - "type=cpu"
      - "level=warning"
    annotations:
      summary: "CPU使用率告警"
      description: "主机 {{ $labels.ident }} CPU使用率超过80%"

  - name: "内存使用率超过80%"
    note: "主机内存使用率超过80%"
    prom_ql: "mem_used_percent > 80"
    severity: 2
    interval: 15
    labels:
      - "type=memory"
```

然后导入：

```bash
./scripts/n9e-setup.sh import my-rules.yaml 1
```

## 预定义告警规则模板

工具内置了常用的告警规则模板：

| 类型 | 说明 | 默认阈值 |
|------|------|----------|
| cpu | CPU 使用率告警 | 80% |
| memory | 内存使用率告警 | 80% |
| disk | 磁盘使用率告警 | 85% |
| host | 主机宕机告警 | - |
| network | 网络错误告警 | - |
| load | 系统负载告警 | 10 |
| diskio | 磁盘 IO 告警 | 80% |
| docker | Docker 容器告警 | - |

使用预设模板：

```bash
# 添加所有预设规则
./scripts/n9e-setup.sh add-preset all 1

# 添加特定类型的规则
./scripts/n9e-setup.sh add-preset cpu 1 90    # CPU 阈值 90%
./scripts/n9e-setup.sh add-preset disk 1 95   # 磁盘阈值 95%
```

## API 参考

### Nightingale API 端点

| 功能 | 方法 | 端点 |
|------|------|------|
| 登录 | POST | `/api/n9e/auth/login` |
| 业务组列表 | GET | `/api/n9e/busi-groups` |
| 创建业务组 | POST | `/api/n9e/busi-groups` |
| 告警规则列表 | GET | `/api/n9e/busi-group/{id}/alert-rules` |
| 创建告警规则 | POST | `/api/n9e/busi-group/{id}/alert-rules` |
| 更新告警规则 | PUT | `/api/n9e/busi-group/{id}/alert-rule/{rid}` |
| 删除告警规则 | DELETE | `/api/n9e/busi-group/{id}/alert-rules` |
| 告警屏蔽列表 | GET | `/api/n9e/busi-group/{id}/alert-mutes` |
| 数据源列表 | POST | `/api/n9e/datasource/list` |
| 监控目标列表 | GET | `/api/n9e/targets` |

### 告警严重程度

| 值 | 级别 | 说明 |
|----|------|------|
| 1 | 紧急 | Emergency - 需要立即处理 |
| 2 | 警告 | Warning - 需要关注 |
| 3 | 通知 | Notice - 一般通知 |

## 高级用法

### 在 Python 代码中使用

```python
from scripts.n9e_alert_config import N9EConfig, N9EClient, AlertRule, AlertRuleTemplates

# 创建客户端
config = N9EConfig.from_env()
client = N9EClient(config)

# 登录
if client.login():
    # 创建业务组
    group_id = client.get_or_create_busi_group("My Group")
    
    # 创建自定义告警规则
    rule = AlertRule(
        name="自定义告警",
        prom_ql='my_metric > 100',
        severity=2
    )
    client.create_alert_rules(group_id, [rule])
    
    # 使用预定义模板
    templates = AlertRuleTemplates.get_all_templates()
    client.create_alert_rules(group_id, templates)
```

### 与 Categraf 集成

1. 部署 Categraf 到目标主机
2. 配置 Categraf 上报地址指向 Nightingale
3. 使用本工具配置告警规则

```bash
# 安装 Categraf
./src/nightingale/scripts/install-categraf.sh

# 配置告警规则
./scripts/n9e-setup.sh full-setup "Production"
```

## 故障排除

### 常见问题

1. **无法连接到 Nightingale**
   - 检查 `NIGHTINGALE_HOST` 和 `NIGHTINGALE_PORT` 配置
   - 确认 Nightingale 服务已启动：`docker-compose ps nightingale`

2. **登录失败**
   - 检查用户名密码是否正确
   - 确认用户有足够权限

3. **导入规则失败**
   - 检查 YAML 格式是否正确
   - 确认业务组 ID 存在

4. **告警未触发**
   - 检查 PromQL 语法是否正确
   - 确认数据源配置正确
   - 检查采集器是否正常上报数据

### 调试模式

启用调试模式查看详细日志：

```bash
DEBUG=true ./scripts/n9e-setup.sh status
```

或者使用 Python 脚本：

```bash
python3 scripts/n9e-alert-config.py --verbose status
```

## 相关文档

- [Nightingale 官方文档](https://n9e.github.io/)
- [Nightingale GitHub](https://github.com/ccfos/nightingale)
- [PromQL 语法指南](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Categraf 文档](https://github.com/flashcatcloud/categraf)
