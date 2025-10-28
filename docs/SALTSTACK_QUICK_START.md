# SaltStack AppHub集成 - 快速指南

## 🎯 完成的工作

### ✅ 已修复

1. **AppHub构建** - SaltStack包现在正确下载和分发（支持多架构）
   - 修复了GitHub下载URL命名格式（DEB用下划线，RPM用连字符）
   - 修复了Alpine Linux仓库版本问题（v3.21 → v3.20）
   - **支持AMD64和ARM64双架构**：同时下载两种架构的包
   - 成功构建包含**28个SaltStack包**（7组件 × 2架构 × 2格式 = 28包）
     * 14个DEB包（7个amd64 + 7个arm64）
     * 14个RPM包（7个x86_64 + 7个aarch64）

2. **Minion安装脚本** - 现在从AppHub下载，不再使用公网
   - 文件: `src/backend/internal/services/saltstack_client_service.go`
   - 自动从AppHub下载salt-common和salt-minion包
   - 支持Ubuntu/Debian和CentOS/RHEL系统

3. **测试套件** - 创建了完整的Playwright E2E测试
   - 文件: `test/e2e/specs/saltstack-integration.spec.js`
   - 脚本: `test/e2e/run-saltstack-tests.sh`
   - 验证AppHub包可用性、状态页面显示、包完整性

## 🚀 快速开始

### 1. 重启AppHub（使用新镜像）

```bash
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 停止旧容器
docker-compose stop apphub

# 删除旧容器
docker-compose rm -f apphub

# 启动新容器（使用新构建的镜像）
docker-compose up -d apphub

# 验证容器运行
docker ps | grep apphub

# 验证SaltStack包
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/saltstack-deb/
```

### 2. 验证AppHub服务

```bash
# 检查包索引
curl -I http://192.168.0.200:53434/pkgs/saltstack-deb/Packages.gz

# 检查不同架构的minion包
curl -I http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_amd64.deb
curl -I http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_arm64.deb

# 验证包数量（应该有28个包）
curl -s http://192.168.0.200:53434/pkgs/saltstack-deb/ | grep -c '\.deb'  # 应该是14
curl -s http://192.168.0.200:53434/pkgs/saltstack-rpm/ | grep -c '\.rpm'  # 应该是14

# 下载测试（根据你的系统架构选择）
curl -O http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_amd64.deb
# 或
curl -O http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_arm64.deb

ls -lh salt-minion_3007.8_*.deb
```

### 3. 运行Playwright测试

```bash
# 赋予执行权限
chmod +x test/e2e/run-saltstack-tests.sh

# 运行测试
./test/e2e/run-saltstack-tests.sh

# 查看HTML报告
npx playwright show-report
```

### 4. 测试Minion安装（手动）

```bash
# SSH到测试节点（先检测架构）
ssh root@192.168.18.154  # 密码: rootpass123

# 检测节点架构
dpkg --print-architecture
# 输出: amd64 或 arm64

# 根据架构下载对应的包
cd /tmp
ARCH=$(dpkg --print-architecture)
echo "Node architecture: $ARCH"

# 下载包（自动选择架构）
wget http://192.168.0.200:53434/pkgs/saltstack-deb/salt-common_3007.8_${ARCH}.deb
wget http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_${ARCH}.deb

# 安装
apt-get update
apt-get install -y python3
dpkg -i salt-common_3007.8_${ARCH}.deb
dpkg -i salt-minion_3007.8_${ARCH}.deb || apt-get install -f -y

# 配置
cat > /etc/salt/minion << EOF
master: 192.168.18.154
id: test-ssh01
EOF

# 启动
systemctl enable salt-minion
systemctl start salt-minion
systemctl status salt-minion
```

### 5. Master端接受Minion

```bash
# 进入SaltStack容器
docker exec -it ai-infra-saltstack bash

# 查看待接受的密钥
salt-key -L

# 接受密钥
salt-key -a test-ssh01

# 测试连接
salt 'test-ssh01' test.ping
```

## 📋 待完成工作

### 高优先级 (本周)

- [ ] 启动AppHub新容器并验证
- [ ] 运行Playwright测试
- [ ] 在test-ssh01/02/03上测试Minion安装
- [ ] 修复Frontend Master信息显示（version, uptime, config path显示"unknown"）

### 中优先级 (下周)

- [ ] 实现SSH自动化测试（完成Playwright SSH测试）
- [ ] 修复API Status显示逻辑
- [ ] 实现Minion列表动态更新

### 低优先级

- [ ] 添加包校验和验证
- [ ] 支持更多架构（amd64, armhf）
- [ ] 优化包下载性能

## 📦 包信息

### SaltStack v3007.8 包列表（多架构支持）

**总计**: 28个包
- 14个DEB包（7组件 × 2架构）
- 14个RPM包（7组件 × 2架构）

**DEB包 - AMD64架构** (7个):

```text
salt-common_3007.8_amd64.deb  (25MB)  - 核心库和依赖
salt-minion_3007.8_amd64.deb (102KB) - Minion客户端
salt-master_3007.8_amd64.deb (114KB) - Master服务端
salt-api_3007.8_amd64.deb    (87KB)  - REST API
salt-cloud_3007.8_amd64.deb  (89KB)  - 云服务集成
salt-ssh_3007.8_amd64.deb    (88KB)  - SSH无代理模式
salt-syndic_3007.8_amd64.deb (87KB)  - Master of Masters
```

**DEB包 - ARM64架构** (7个):

```text
salt-common_3007.8_arm64.deb  (25MB)  - 核心库和依赖
salt-minion_3007.8_arm64.deb (102KB) - Minion客户端
salt-master_3007.8_arm64.deb (114KB) - Master服务端
salt-api_3007.8_arm64.deb    (87KB)  - REST API
salt-cloud_3007.8_arm64.deb  (89KB)  - 云服务集成
salt-ssh_3007.8_arm64.deb    (88KB)  - SSH无代理模式
salt-syndic_3007.8_arm64.deb (87KB)  - Master of Masters
```

**RPM包 - x86_64架构** (7个):

```text
salt-3007.8-0.x86_64.rpm
salt-minion-3007.8-0.x86_64.rpm
salt-master-3007.8-0.x86_64.rpm
salt-api-3007.8-0.x86_64.rpm
salt-cloud-3007.8-0.x86_64.rpm
salt-ssh-3007.8-0.x86_64.rpm
salt-syndic-3007.8-0.x86_64.rpm
```

**RPM包 - aarch64架构** (7个):

```text
salt-3007.8-0.aarch64.rpm
salt-minion-3007.8-0.aarch64.rpm
salt-master-3007.8-0.aarch64.rpm
salt-api-3007.8-0.aarch64.rpm
salt-cloud-3007.8-0.aarch64.rpm
salt-ssh-3007.8-0.aarch64.rpm
salt-syndic-3007.8-0.aarch64.rpm
```

**架构自动检测**: Backend安装脚本会自动检测节点架构（dpkg --print-architecture / uname -m）并下载对应架构的包。

## 🔗 访问地址

- **AppHub**: http://192.168.0.200:53434
- **SaltStack Web UI**: http://192.168.18.154:8080/slurm
- **包目录**: http://192.168.0.200:53434/pkgs/saltstack-deb/
- **包索引**: http://192.168.0.200:53434/pkgs/saltstack-deb/Packages.gz

## 📚 相关文档

- 详细报告: `docs/SALTSTACK_APPHUB_INTEGRATION.md`
- Playwright测试: `test/e2e/specs/saltstack-integration.spec.js`
- 测试脚本: `test/e2e/run-saltstack-tests.sh`
- AppHub Dockerfile: `src/apphub/Dockerfile`
- Backend安装服务: `src/backend/internal/services/saltstack_client_service.go`

## 🐛 已知问题

1. **Master信息显示"unknown"** (Frontend/Backend)
   - 影响: 版本、启动时间、配置路径显示为unknown
   - 状态: 待修复
   - 优先级: 高

2. **Minion计数显示0** (Frontend)
   - 影响: 即使有Minion连接也显示0
   - 状态: 待验证
   - 优先级: 高

3. **API Status显示disconnected** (Frontend)
   - 影响: Salt API状态检测逻辑有问题
   - 状态: 待修复
   - 优先级: 中

## ✅ 验证清单

使用此清单验证所有功能：

```text
□ AppHub容器成功启动
□ 访问http://192.168.0.200:53434可以看到欢迎页
□ /pkgs/saltstack-deb/目录列出14个DEB包（7个amd64 + 7个arm64）
□ /pkgs/saltstack-rpm/目录列出14个RPM包（7个x86_64 + 7个aarch64）
□ Packages.gz索引文件可以下载
□ Playwright测试全部通过
□ 可以根据节点架构自动下载对应的包
□ test-ssh01可以从AppHub下载并安装Minion（正确的架构）
□ Minion成功连接到Master
□ salt 'test-ssh01' test.ping返回True
□ Frontend显示Minion计数为1（或3，如果安装了3个节点）
□ 支持混合架构环境（AMD64和ARM64节点可以共存）
```

## 🆘 故障排除

### AppHub包不可访问

```bash
# 检查容器日志
docker logs ai-infra-apphub

# 检查nginx配置
docker exec ai-infra-apphub cat /etc/nginx/nginx.conf

# 检查包目录
docker exec ai-infra-apphub ls -lR /usr/share/nginx/html/pkgs/
```

### Minion安装失败

```bash
# 检查网络连通性
ping 192.168.0.200

# 手动下载测试
wget http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_arm64.deb

# 检查依赖
dpkg -i salt-minion_3007.8_arm64.deb  # 会显示缺少的依赖
apt-get install -f  # 修复依赖
```

### Minion无法连接Master

```bash
# 检查Master地址
cat /etc/salt/minion | grep master

# 检查防火墙
sudo ufw status
# Salt使用端口4505和4506

# 查看Minion日志
journalctl -u salt-minion -f

# 手动测试连接
salt-minion -l debug
```

---

**最后更新**: 2024-10-28 15:15  
**状态**: ✅ 构建完成，待测试验证
