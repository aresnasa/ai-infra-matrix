# Salt Docker Infrastructure

这是一个完整的 SaltStack 基础设施，使用 Docker Compose 构建，包含 1 个 Master 和 3 个 Minion，以及完整的自动化测试套件。

## 🏗️ 架构概览

```
┌─────────────────┐    ┌─────────────────┐
│   Salt Master   │    │  Test Runner    │
│  (salt-master)  │    │ (salt-test-runner)│
└─────────────────┘    └─────────────────┘
         │                       │
    ┌────┴────┐                 │
    │ Port    │                 │
    │ 4505    │                 │
    │ 4506    │                 │
    └─────────┘                 │
         │                       │
    ┌────┴─────────────────────┐ │
    │     Salt Network         │ │
    │   (172.20.0.0/16)       │ │
    └─┬─────────┬─────────┬────┘ │
      │         │         │      │
┌─────▼───┐ ┌───▼───┐ ┌───▼────┐ │
│Minion-1 │ │Minion-2│ │Minion-3│ │
│Frontend │ │Backend │ │Database│ │
└─────────┘ └───────┘ └────────┘ │
                                 │
         ┌───────────────────────┘
         │
    ┌────▼────┐
    │  Tests  │
    │ Network │
    └─────────┘
```

## 🚀 快速开始

### 1. 启动基础设施
```bash
# 一键启动所有服务
./start.sh

# 或者使用管理脚本
./salt-manager.sh start
```

### 2. 检查状态
```bash
# 查看所有服务状态
./salt-manager.sh status

# 查看容器状态
docker-compose ps
```

### 3. 运行测试
```bash
# 运行完整测试套件
./run-tests-full.sh

# 或者使用管理脚本
./salt-manager.sh test
```

## 📋 管理命令

使用 `salt-manager.sh` 脚本进行日常管理：

```bash
# 基础操作
./salt-manager.sh start      # 启动基础设施
./salt-manager.sh stop       # 停止基础设施
./salt-manager.sh restart    # 重启基础设施
./salt-manager.sh status     # 查看状态

# 日志管理
./salt-manager.sh logs                # 查看所有日志
./salt-manager.sh logs salt-master    # 查看特定服务日志

# 测试和验证
./salt-manager.sh test       # 运行完整测试套件
./salt-manager.sh keys       # 查看 Salt 密钥状态
./salt-manager.sh apply      # 应用 Salt 状态

# 调试和管理
./salt-manager.sh shell minion-1     # 进入 minion 容器
./salt-manager.sh exec "salt '*' test.ping"  # 执行 Salt 命令

# 维护操作
./salt-manager.sh clean      # 清理所有容器和卷
./salt-manager.sh rebuild    # 重新构建基础设施
```

## 🧪 测试套件

### 自动化测试包括：

1. **连接性测试**
   - Master 容器健康检查
   - Minion 连接验证
   - 网络通信测试

2. **配置一致性测试**
   - 密钥接受验证
   - Grains 配置检查
   - Pillar 数据验证
   - 状态应用测试

3. **性能测试**
   - 命令执行时间
   - 网络延迟测试

4. **配置漂移检测**
   - 状态一致性验证
   - 配置变更检测

### 测试报告

测试完成后会生成详细的 HTML 报告，包含：
- 容器状态
- 网络配置
- Grains 信息
- 性能指标

## 🔧 配置说明

### Master 配置
- **端口**: 4505 (Publisher), 4506 (Request Server)
- **自动接受密钥**: 启用（仅用于测试环境）
- **日志级别**: info
- **工作线程**: 5

### Minion 配置
- **minion-1**: Frontend 角色，包含 nginx
- **minion-2**: Backend 角色，包含 python3
- **minion-3**: Database 角色，包含 sqlite

### 网络配置
- **网络**: bridge 模式
- **子网**: 172.20.0.0/16
- **DNS**: 自动解析容器名

## 📁 目录结构

```
docker-saltstack/
├── docker-compose.yml          # 主要的 compose 配置
├── Dockerfile.master           # Master 容器构建文件
├── Dockerfile.minion           # Minion 容器构建文件
├── Dockerfile.test             # 测试容器构建文件
├── start.sh                    # 启动脚本
├── stop.sh                     # 停止脚本
├── salt-manager.sh             # 管理脚本
├── run-tests-full.sh           # 完整测试脚本
├── salt-config/                # Salt 配置文件
│   ├── master/
│   │   └── master.conf
│   ├── minion-1/
│   │   └── minion.conf
│   ├── minion-2/
│   │   └── minion.conf
│   ├── minion-3/
│   │   └── minion.conf
│   └── minion-template/
│       └── minion.conf
├── salt-states/                # Salt 状态文件
│   ├── top.sls
│   ├── common.sls
│   ├── database.sls
│   └── web/
│       ├── frontend.sls
│       └── backend.sls
├── salt-pillar/                # Salt Pillar 数据
│   ├── top.sls
│   └── common.sls
├── scripts/                    # 辅助脚本
│   ├── start-master.sh
│   ├── start-minion.sh
│   ├── master-healthcheck.sh
│   └── run-tests.sh
└── tests/                      # 测试文件
    └── test_salt_infrastructure.py
```

## 🔒 安全注意事项

⚠️ **重要提醒**：此配置仅适用于开发和测试环境！

生产环境需要考虑：
- 禁用自动密钥接受
- 配置 SSL/TLS 加密
- 设置防火墙规则
- 使用密钥文件认证
- 配置 RBAC（基于角色的访问控制）

## 🐛 故障排除

### 常见问题

1. **容器启动失败**
   ```bash
   # 检查日志
   ./salt-manager.sh logs
   
   # 重新构建
   ./salt-manager.sh rebuild
   ```

2. **Minion 无法连接 Master**
   ```bash
   # 检查网络
   docker network inspect docker-saltstack_salt-network
   
   # 检查密钥状态
   ./salt-manager.sh keys
   ```

3. **测试失败**
   ```bash
   # 查看详细测试输出
   ./run-tests-full.sh
   
   # 手动测试连接
   ./salt-manager.sh exec "salt '*' test.ping"
   ```

### 清理和重置

```bash
# 完全清理
./salt-manager.sh clean

# 重新开始
./salt-manager.sh start
```

## 📊 监控和日志

### 实时监控
```bash
# 监控所有服务
./salt-manager.sh logs

# 监控特定服务
./salt-manager.sh logs salt-master
./salt-manager.sh logs salt-minion-1
```

### 性能指标
```bash
# 查看容器资源使用
docker stats

# 查看网络流量
docker network inspect docker-saltstack_salt-network
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目！

## 📝 许可证

本项目基于 MIT 许可证开源。
Docker Compose setup to spin up a salt master and minions.

You can read a full article describing how to use this setup [here](https://medium.com/@timlwhite/the-simplest-way-to-learn-saltstack-cd9f5edbc967).

You will need a system with Docker and Docker Compose installed to use this project.

Just run:

`docker-compose up`

from a checkout of this directory, and the master and minion will start up with debug logging to the console.

Then you can run (in a separate shell window):

`docker-compose exec salt-master bash`

and it will log you into the command line of the salt-master server.

From that command line you can run something like:

`salt '*' test.ping`

and in the window where you started docker compose, you will see the log output of both the master sending the command and the minion receiving the command and replying.

[The Salt Remote Execution Tutorial](https://docs.saltstack.com/en/latest/topics/tutorials/modules.html) has some quick examples of the comamnds you can run from the master.

Note: you will see log messages like : "Could not determine init system from command line" - those are just because salt is running in the foreground and not from an auto-startup.

The salt-master is set up to accept all minions that try to connect.  Since the network that the salt-master sees is only the docker-compose network, this means that only minions within this docker-compose service network will be able to connect (and not random other minions external to docker).

#### Running multiple minions:

`docker-compose up --scale salt-minion=2`

This will start up two minions instead of just one.

#### Host Names
The **hostnames** match the names of the containers - so the master is `salt-master` and the minion is `salt-minion`.

If you are running more than one minion with `--scale=2`, you will need to use `docker-saltstack_salt-minion_1` and `docker-saltstack_salt-minion_2` for the minions if you want to target them individually.
