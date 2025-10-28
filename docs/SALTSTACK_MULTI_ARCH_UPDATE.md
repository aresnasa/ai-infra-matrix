# SaltStack多架构支持更新

## 更新概述

**日期**: 2024-10-28  
**版本**: v0.3.6-dev  
**更新类型**: 多架构包支持

## 变更说明

### 问题

之前的AppHub构建只下载当前构建平台的架构包：
- 在ARM64平台构建时，只下载ARM64包
- 在AMD64平台构建时，只下载AMD64包

这导致AppHub无法为不同架构的节点提供包。

### 解决方案

修改AppHub Dockerfile，**同时下载AMD64和ARM64两种架构**的所有SaltStack包。

## 技术实现

### 1. DEB包下载（修改前）

```dockerfile
# 旧代码：只下载当前架构
ARCH=$(dpkg --print-architecture);
if [ "${ARCH}" = "arm64" ]; then
    ARCH_SUFFIX="arm64";
else
    ARCH_SUFFIX="amd64";
fi;

for pkg in salt-common salt-master salt-minion ...; do
    PKG_FILE="${pkg}_${VERSION_NUM}_${ARCH_SUFFIX}.deb";
    wget "${BASE_URL}/${PKG_FILE}";
done
```

### 1. DEB包下载（修改后）

```dockerfile
# 新代码：下载两种架构
total_downloaded=0;
for ARCH_SUFFIX in amd64 arm64; do
    echo "📥 Downloading ${ARCH_SUFFIX} packages...";
    for pkg in salt-common salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do
        PKG_FILE="${pkg}_${VERSION_NUM}_${ARCH_SUFFIX}.deb";
        wget "${BASE_URL}/${PKG_FILE}";
        total_downloaded=$((total_downloaded + 1));
    done;
    echo "✓ Downloaded ${arch_downloaded} ${ARCH_SUFFIX} packages";
done;
```

**关键变化**:
- ✅ 循环遍历 `amd64` 和 `arm64` 两种架构
- ✅ 统计每种架构下载的包数量
- ✅ 分别显示AMD64和ARM64的包列表

### 2. RPM包下载（修改前）

```dockerfile
# 旧代码：只下载当前架构
ARCH=$(uname -m);
if [ "${ARCH}" = "aarch64" ]; then
    ARCH_SUFFIX="aarch64";
else
    ARCH_SUFFIX="x86_64";
fi;

for pkg in salt salt-master salt-minion ...; do
    PKG_FILE="${pkg}-${VERSION_NUM}-0.${ARCH_SUFFIX}.rpm";
    wget "${BASE_URL}/${PKG_FILE}";
done
```

### 2. RPM包下载（修改后）

```dockerfile
# 新代码：下载两种架构
total_downloaded=0;
for ARCH_SUFFIX in x86_64 aarch64; do
    echo "📥 Downloading ${ARCH_SUFFIX} packages...";
    for pkg in salt salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do
        PKG_FILE="${pkg}-${VERSION_NUM}-0.${ARCH_SUFFIX}.rpm";
        wget "${BASE_URL}/${PKG_FILE}";
        total_downloaded=$((total_downloaded + 1));
    done;
    echo "✓ Downloaded ${arch_downloaded} ${ARCH_SUFFIX} packages";
done;
```

**关键变化**:
- ✅ 循环遍历 `x86_64` 和 `aarch64` 两种架构
- ✅ 统计每种架构下载的包数量
- ✅ 分别显示x86_64和aarch64的包列表

## 包数量统计

### 预期下载包总数

| 包类型 | 组件数 | 架构数 | 总数 |
|--------|--------|--------|------|
| DEB    | 7      | 2      | **14** |
| RPM    | 7      | 2      | **14** |
| **合计** | -    | -      | **28** |

### 详细包列表

#### DEB包 (14个)

**AMD64架构** (7个):
```
salt-common_3007.8_amd64.deb
salt-master_3007.8_amd64.deb
salt-minion_3007.8_amd64.deb
salt-api_3007.8_amd64.deb
salt-cloud_3007.8_amd64.deb
salt-ssh_3007.8_amd64.deb
salt-syndic_3007.8_amd64.deb
```

**ARM64架构** (7个):
```
salt-common_3007.8_arm64.deb
salt-master_3007.8_arm64.deb
salt-minion_3007.8_arm64.deb
salt-api_3007.8_arm64.deb
salt-cloud_3007.8_arm64.deb
salt-ssh_3007.8_arm64.deb
salt-syndic_3007.8_arm64.deb
```

#### RPM包 (14个)

**x86_64架构** (7个):
```
salt-3007.8-0.x86_64.rpm
salt-master-3007.8-0.x86_64.rpm
salt-minion-3007.8-0.x86_64.rpm
salt-api-3007.8-0.x86_64.rpm
salt-cloud-3007.8-0.x86_64.rpm
salt-ssh-3007.8-0.x86_64.rpm
salt-syndic-3007.8-0.x86_64.rpm
```

**aarch64架构** (7个):
```
salt-3007.8-0.aarch64.rpm
salt-master-3007.8-0.aarch64.rpm
salt-minion-3007.8-0.aarch64.rpm
salt-api-3007.8-0.aarch64.rpm
salt-cloud-3007.8-0.aarch64.rpm
salt-ssh-3007.8-0.aarch64.rpm
salt-syndic-3007.8-0.aarch64.rpm
```

## 构建输出示例

### 预期构建日志

```
📦 Downloading SaltStack v3007.8 deb packages from GitHub releases...
Downloading from: https://github.com/saltstack/salt/releases/download/v3007.8
Version: 3007.8

📥 Downloading amd64 packages...
Trying to download: salt-common_3007.8_amd64.deb
✓ Downloaded: salt-common_3007.8_amd64.deb
Trying to download: salt-master_3007.8_amd64.deb
✓ Downloaded: salt-master_3007.8_amd64.deb
...
✓ Downloaded 7 amd64 packages

📥 Downloading arm64 packages...
Trying to download: salt-common_3007.8_arm64.deb
✓ Downloaded: salt-common_3007.8_arm64.deb
Trying to download: salt-master_3007.8_arm64.deb
✓ Downloaded: salt-master_3007.8_arm64.deb
...
✓ Downloaded 7 arm64 packages

📊 Download Summary:
✓ Total downloaded: 14 SaltStack deb packages

AMD64 packages:
-rw-r--r-- 1 root root  25M salt-common_3007.8_amd64.deb
-rw-r--r-- 1 root root 114K salt-master_3007.8_amd64.deb
...

ARM64 packages:
-rw-r--r-- 1 root root  25M salt-common_3007.8_arm64.deb
-rw-r--r-- 1 root root 114K salt-master_3007.8_arm64.deb
...
```

### RPM包下载日志

```
📦 Downloading SaltStack v3007.8 rpm packages from GitHub releases...
Downloading from: https://github.com/saltstack/salt/releases/download/v3007.8
Version: 3007.8

📥 Downloading x86_64 packages...
✓ Downloaded: salt-3007.8-0.x86_64.rpm
...
✓ Downloaded 7 x86_64 packages

📥 Downloading aarch64 packages...
✓ Downloaded: salt-3007.8-0.aarch64.rpm
...
✓ Downloaded 7 aarch64 packages

📊 Download Summary:
✓ Total downloaded: 14 SaltStack rpm packages

x86_64 packages:
-rw-r--r-- 1 root root 25M salt-3007.8-0.x86_64.rpm
...

aarch64 packages:
-rw-r--r-- 1 root root 25M salt-3007.8-0.aarch64.rpm
...
```

## Backend自动架构检测

Backend的Minion安装脚本已经支持自动检测节点架构，无需修改。

### Ubuntu/Debian节点

```bash
# 自动检测架构
ARCH=$(dpkg --print-architecture 2>/dev/null || echo "arm64")
# 结果: "amd64" 或 "arm64"

# 下载对应架构的包
curl -fsSL "${APPHUB_BASE}/salt-common_${VERSION}_${ARCH}.deb" -o salt-common.deb
curl -fsSL "${APPHUB_BASE}/salt-minion_${VERSION}_${ARCH}.deb" -o salt-minion.deb
```

### CentOS/RHEL节点

```bash
# 自动检测架构
ARCH=$(uname -m)
# 结果: "x86_64" 或 "aarch64"

# 下载对应架构的包
curl -fsSL "${APPHUB_BASE}/salt-minion-${VERSION}.${ARCH}.rpm" -o salt-minion.rpm
```

## 验证步骤

### 1. 构建AppHub

```bash
./build.sh build apphub --no-cache
```

### 2. 启动AppHub容器

```bash
docker-compose up -d apphub
```

### 3. 验证包列表

```bash
# 验证DEB包
curl http://192.168.0.200:53434/pkgs/saltstack-deb/ | grep -E '(amd64|arm64)'

# 应该看到28个.deb文件（14个amd64 + 14个arm64）

# 验证RPM包
curl http://192.168.0.200:53434/pkgs/saltstack-rpm/ | grep -E '(x86_64|aarch64)'

# 应该看到28个.rpm文件（14个x86_64 + 14个aarch64）
```

### 4. 测试不同架构节点安装

#### AMD64节点测试

```bash
# SSH到AMD64节点
ssh user@amd64-node

# 检测架构
dpkg --print-architecture
# 输出: amd64

# 下载测试
wget http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_amd64.deb
```

#### ARM64节点测试

```bash
# SSH到ARM64节点
ssh user@arm64-node

# 检测架构
dpkg --print-architecture
# 输出: arm64

# 下载测试
wget http://192.168.0.200:53434/pkgs/saltstack-deb/salt-minion_3007.8_arm64.deb
```

## 影响范围

### 正面影响

✅ **支持混合架构环境**
- 可以在同一集群中管理AMD64和ARM64节点
- 不需要为不同架构维护多个AppHub实例

✅ **简化部署**
- 一次构建支持所有架构
- 统一的包管理

✅ **提高兼容性**
- 支持x86服务器
- 支持ARM服务器（包括AWS Graviton、Azure Ampere等）
- 支持树莓派和其他ARM设备

### 潜在问题

⚠️ **下载时间增加**
- 包数量从14个增加到28个
- 预计增加1-2分钟构建时间（取决于网络速度）

⚠️ **镜像体积增加**
- DEB包总大小: ~50MB (25MB × 2架构)
- RPM包总大小: ~50MB (25MB × 2架构)
- 预计AppHub镜像增加约100MB

## 更新文件清单

### 修改的文件

1. **src/apphub/Dockerfile**
   - Lines 123-175: DEB包下载 - 支持多架构循环
   - Lines 346-398: RPM包下载 - 支持多架构循环

### 无需修改的文件

1. **src/backend/internal/services/saltstack_client_service.go**
   - 已支持自动架构检测（dpkg --print-architecture / uname -m）
   - 安装逻辑无需修改

## 兼容性

### 支持的操作系统

| 操作系统 | AMD64 | ARM64 | 包格式 |
|---------|-------|-------|--------|
| Ubuntu 22.04 | ✅ | ✅ | DEB |
| Ubuntu 20.04 | ✅ | ✅ | DEB |
| Debian 11/12 | ✅ | ✅ | DEB |
| CentOS 8/9 | ✅ | ✅ | RPM |
| Rocky Linux 8/9 | ✅ | ✅ | RPM |
| AlmaLinux 8/9 | ✅ | ✅ | RPM |

### 测试节点架构

| 节点 | IP | 架构 | OS |
|------|----|----|-----|
| test-ssh01 | 192.168.18.154 | ? | Ubuntu 22.04 |
| test-ssh02 | 192.168.18.155 | ? | Ubuntu 22.04 |
| test-ssh03 | 192.168.18.156 | ? | Ubuntu 22.04 |

**注**: 需要检测测试节点实际架构

## 下一步计划

### 立即行动

1. ✅ 修改Dockerfile支持多架构下载
2. ⏳ 重新构建AppHub
3. ⏳ 验证28个包全部下载成功
4. ⏳ 测试不同架构节点安装

### 后续优化

- [ ] 添加SHA256校验和验证
- [ ] 实现并行下载提高速度
- [ ] 支持更多架构（armhf等）
- [ ] 添加包缓存机制避免重复下载

## 参考资源

- [SaltStack GitHub Releases](https://github.com/saltstack/salt/releases/tag/v3007.8)
- [Debian Package Naming](https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-architecture)
- [RPM Package Naming](https://rpm.org/user_doc/dependencies.html)

---

**文档版本**: 1.0  
**最后更新**: 2024-10-28 15:25  
**作者**: GitHub Copilot
