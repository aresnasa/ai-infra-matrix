# SafeLine WAF Component
# 雷池 Web 应用防火墙 - 长亭科技开源项目
#
# 官方仓库: https://github.com/chaitin/SafeLine
# 官方文档: https://docs.safeline.chaitin.cn/

## 组件说明

SafeLine 是一款开源的 Web 应用防火墙 (WAF)，由长亭科技开发维护。
本项目将 SafeLine 集成为最外层的安全防护，为整个 AI Infrastructure Matrix 提供安全保障。

## 首次启动前准备

在启动 SafeLine 服务前，需要创建必要的数据目录：

```bash
# 推荐方式: 使用 build.sh 命令 (自动设置安全权限)
./build.sh init-safeline
```

或者手动创建目录：

```bash
# 创建 SafeLine 数据目录
mkdir -p ./data/safeline/{resources,logs,run}
mkdir -p ./data/safeline/resources/{postgres/data,mgt,sock,nginx,detector,chaos,cache,luigi}
mkdir -p ./data/safeline/logs/{nginx,detector}

# 设置安全的目录权限 (不要使用 chmod 777!)
chmod 755 ./data/safeline
chmod 755 ./data/safeline/resources
chmod 700 ./data/safeline/resources/postgres  # 数据库数据需要严格权限
chmod 750 ./data/safeline/resources/sock      # Socket 目录
chmod 750 ./data/safeline/run                 # 运行时目录
```

## 服务组件

| 服务 | 容器名 | 说明 |
|------|--------|------|
| safeline-postgres | safeline-pg | SafeLine 专用 PostgreSQL 数据库 |
| safeline-mgt | safeline-mgt | 管理控制台 (Web UI) |
| safeline-detector | safeline-detector | 检测引擎 |
| safeline-tengine | safeline-tengine | 反向代理 (基于 Tengine, host 网络模式) |
| safeline-luigi | safeline-luigi | 任务调度器 |
| safeline-fvm | safeline-fvm | 特征验证模块 |
| safeline-chaos | safeline-chaos | 混沌工程模块 |

## 网络架构

```
                    ┌─────────────────┐
                    │   Internet      │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ safeline-tengine│ (host 网络, 监听 80/443)
                    │   (WAF 入口)    │
                    └────────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │      safeline-ce 网络        │
              │   (172.22.222.0/24)         │
              │                              │
              │  ┌──────────────────────┐   │
              │  │  safeline-mgt (.4)   │   │
              │  │  safeline-detector   │   │
              │  │  safeline-postgres   │   │
              │  │  ...                 │   │
              │  └──────────────────────┘   │
              │                              │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │    ai-infra-network         │
              │   (项目其他服务)             │
              └─────────────────────────────┘
```

## 环境变量

在 `.env` 文件中配置以下变量:

```bash
# SafeLine WAF Configuration
SAFELINE_DIR=./data/safeline           # 数据目录
SAFELINE_IMAGE_TAG=9.3.0               # 镜像版本
SAFELINE_MGT_PORT=9443                 # 管理控制台端口
SAFELINE_POSTGRES_PASSWORD=xxx         # 数据库密码
SAFELINE_SUBNET_PREFIX=172.22.222      # 内部网络子网
SAFELINE_IMAGE_PREFIX=chaitin          # 镜像前缀
SAFELINE_ARCH_SUFFIX=                  # 架构后缀 (ARM: -arm)
SAFELINE_REGION=                       # 区域后缀 (可选)
```

## 架构支持

- **x86_64/amd64**: 使用默认镜像，`SAFELINE_ARCH_SUFFIX` 留空
- **ARM/aarch64**: 设置 `SAFELINE_ARCH_SUFFIX=-arm`

架构后缀由 `build.sh` 自动检测。

## 访问方式

- 管理控制台: `https://<host>:9443`
- 首次登录需要重置管理员密码

## 网络架构

```
                    ┌─────────────────┐
    Internet ───────│  SafeLine WAF   │
                    │  (tengine:80/443)│
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   ai-infra-     │
                    │    network      │
                    │                 │
                    │  ┌───────────┐  │
                    │  │   nginx   │  │
                    │  └───────────┘  │
                    │        │        │
            ┌───────┴────────┴────────┴───────┐
            │                                  │
     ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
     │  frontend   │  │   backend   │  │ jupyterhub  │
     └─────────────┘  └─────────────┘  └─────────────┘
```

## 相关文件

- `docker-compose.yml.tpl`: SafeLine 服务定义
- `config/images.yaml`: 镜像配置
- `.env.example`: 环境变量示例
