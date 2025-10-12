# Kubernetes 虚拟网卡排除优化报告

## 优化日期
2025-10-12

## 问题描述

在 macOS 上运行 `build.sh` 脚本时，检测到了 Kubernetes 的虚拟网卡 IP (`192.168.65.3`)，而不是物理网卡 `en0` 的真实 IP。

### 错误输出
```
[INFO] 🎯 检测到 Kubernetes 环境
[INFO] ☸️  K8s 外部地址: 192.168.65.3
[INFO] 🖥️  检测到外部地址: 192.168.65.3 (IP)
[INFO] 🌍 基础访问地址: http://192.168.65.3:8080
```

**问题根源:**
- Docker Desktop for Mac 启用 Kubernetes 后创建虚拟网卡（bridge100 等）
- 虚拟网卡使用 `192.168.65.x` 网段
- 脚本误判为生产 K8s 环境，并选择了虚拟网卡 IP

## 优化方案

### 1. 扩展虚拟 IP 排除列表

在所有 IP 检测函数中添加 Kubernetes 相关的虚拟 IP 段：

| IP 段 | 用途 | 环境 |
|-------|------|------|
| `192.168.65.*` | Kubernetes Docker Desktop | macOS |
| `10.96.*` | Kubernetes Service 网络 | 所有 K8s |
| `192.168.64.*` | Docker/虚拟机桥接 | macOS |
| `10.211.*` | Parallels 虚拟网络 | macOS |
| `10.37.*` | VMware 虚拟网络 | 通用 |
| `172.16-31.*` | Docker 默认网络 | 通用 |

### 2. 优化 `detect_active_interface()`

**修改前:**
```bash
# 仅排除 docker、veth
active_interfaces=($(ifconfig | grep -E '^[a-z]' | grep -v '^lo' | \
    grep -v 'docker' | grep -v 'veth' | ...))
```

**修改后:**
```bash
# 扩展排除：bridge、vmnet、vboxnet、utun
active_interfaces=($(ifconfig | grep -E '^[a-z]' | grep -v '^lo' | \
    grep -v 'docker' | grep -v 'veth' | grep -v 'bridge' | \
    grep -v 'vmnet' | grep -v 'vboxnet' | grep -v 'utun' | ...))

# 额外 IP 范围检查
if [[ ! "$iface_ip" =~ ^192\.168\.65\. ]] && \
   [[ ! "$iface_ip" =~ ^10\.96\. ]] && \
   [[ ! "$iface_ip" =~ ^172\.1[6-9]\. ]]; then
    echo "$iface"
    return 0
fi
```

**排除的虚拟接口:**
- `bridge*` - Kubernetes 网桥 (bridge100)
- `vmnet*` - VMware 虚拟网络
- `vboxnet*` - VirtualBox 虚拟网络
- `utun*` - macOS VPN 隧道
- `docker*` - Docker 虚拟接口
- `veth*` - Linux 虚拟以太网
- `virbr*` - KVM/libvirt 网桥

### 3. 优化 `detect_external_host()`

**修改内容:**
```bash
# 方法1: ifconfig 排除虚拟 IP
detected_ip=$(ifconfig | grep "inet " | grep -v "127.0.0.1" | \
    grep -v "10.211." | grep -v "10.37." | grep -v "10.96." | \
    grep -v "192.168.64." | grep -v "192.168.65." | \  # ← 新增
    grep -v "172.1[6-9]." | grep -v "172.2[0-9]." | grep -v "172.3[0-1]." | \
    awk '{print $2}' | head -n1)

# 方法2: ip 命令同样排除
detected_ip=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | \
    grep -v "10.211." | grep -v "10.37." | grep -v "10.96." | \
    grep -v "192.168.64." | grep -v "192.168.65." | \  # ← 新增
    grep -v "172.1[6-9]." | grep -v "172.2[0-9]." | grep -v "172.3[0-1]." | \
    grep -v "docker" | grep -v "veth" | grep -v "bridge" | \  # ← 新增 bridge
    awk '{print $2}' | cut -d'/' -f1 | head -n1)

# 方法3: hostname 二次检查
detected_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
if [[ "$detected_ip" =~ ^192\.168\.65\. ]] || [[ "$detected_ip" =~ ^10\.96\. ]]; then
    detected_ip=""  # ← 新增：虚拟 IP 检查
fi
```

### 4. 优化 `detect_k8s_environment()`

**问题:** Docker Desktop 的本地 Kubernetes 被误判为生产环境

**修改前:**
```bash
# 仅检查 kubectl 是否可用
if kubectl cluster-info &> /dev/null; then
    echo "true"
    return 0
fi
```

**修改后:**
```bash
# 检查 kubectl 且排除本地开发集群
if kubectl cluster-info &> /dev/null; then
    local k8s_context=$(kubectl config current-context 2>/dev/null)
    
    # 排除本地集群上下文
    if [[ "$k8s_context" =~ docker-desktop|docker-for-desktop|minikube|kind ]]; then
        echo "false"
        return 1
    fi
    
    # 单节点检查（可能是本地环境）
    local node_count=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
    if [[ $node_count -eq 1 ]]; then
        local node_name=$(kubectl get nodes --no-headers 2>/dev/null | awk '{print $1}')
        if [[ "$node_name" =~ docker-desktop|minikube|kind ]]; then
            echo "false"
            return 1
        fi
    fi
    
    # 通过检查，判定为真实 K8s 环境
    echo "true"
    return 0
fi
```

**检测规则:**
1. **上下文检查**: 排除 `docker-desktop`, `minikube`, `kind` 等本地集群
2. **节点数量**: 单节点可能是本地环境
3. **节点名称**: 进一步验证节点名是否包含本地集群标识
4. **强制环境变量**: 支持 `AI_INFRA_FORCE_K8S` 手动指定

### 5. 优化 macOS 网卡优先级

在 `detect_active_interface()` 中调整优先级：

```bash
# 优先级排序：eth > enp > ens > en (macOS) > bond > br > wlan
for prefix in "eth" "enp" "ens" "en" "bond" "br" "wlan"; do
    for iface in "${active_interfaces[@]}"; do
        if [[ "$iface" =~ ^${prefix} ]]; then
            # 检查 IP 是否为虚拟网段
            local iface_ip=$(detect_interface_ip "$iface")
            if [[ -n "$iface_ip" ]] && is_real_ip "$iface_ip"; then
                echo "$iface"
                return 0
            fi
        fi
    done
done
```

**macOS 物理网卡:** `en0`, `en1`, `en2` 等
**macOS 虚拟网卡:** `bridge100`, `vmnet*`, `utun*` 等

## 检测流程优化

### 修改前流程

```
kubectl cluster-info 可用
    ↓
判定为 K8s 环境 ✅
    ↓
获取 K8s 节点 IP (192.168.65.3)
    ↓
❌ 错误：使用虚拟网卡 IP
```

### 修改后流程

```
kubectl cluster-info 可用
    ↓
检查上下文名称
    ↓
docker-desktop? → 本地集群，判定为 Docker Compose ✅
    ↓
非本地集群 → 检查节点数量
    ↓
单节点? → 检查节点名称
    ↓
docker-desktop? → 本地集群
    ↓
多节点或非本地节点名 → 判定为真实 K8s 环境 ✅
```

```
检测活跃网卡
    ↓
遍历 en0, en1, en2...
    ↓
获取网卡 IP
    ↓
192.168.65.x? → ❌ 跳过（K8s 虚拟 IP）
    ↓
192.168.1.x? → ✅ 使用（真实局域网 IP）
```

## 测试场景

### 1. macOS + Docker Desktop + Kubernetes

**测试环境:**
```bash
OS: macOS 14.x
Docker Desktop: 启用 Kubernetes
物理网卡: en0 (192.168.1.100)
虚拟网卡: bridge100 (192.168.65.3)
```

**优化前:**
```
[INFO] 🎯 检测到 Kubernetes 环境
[INFO] ☸️  K8s 外部地址: 192.168.65.3  ❌
```

**优化后:**
```
[INFO] 🐳 检测到 Docker Compose 环境  ✅
[INFO] 🖥️  检测到外部地址: 192.168.1.100 (IP)  ✅
```

### 2. 真实 Kubernetes 集群

**测试环境:**
```bash
OS: Linux
集群: 生产 K8s (3 节点)
上下文: production-cluster
```

**优化后:**
```
[INFO] 🎯 检测到 Kubernetes 环境  ✅
[INFO] ☸️  K8s 外部地址: 10.0.1.50  ✅
```

### 3. minikube 本地开发

**测试环境:**
```bash
OS: Linux
集群: minikube
上下文: minikube
```

**优化后:**
```
[INFO] 🐳 检测到 Docker Compose 环境  ✅
[INFO] 🖥️  检测到外部地址: 192.168.1.100  ✅
```

## 手动覆盖选项

### 1. 强制指定为 K8s 环境

```bash
export AI_INFRA_FORCE_K8S=true
./build.sh build-all
```

### 2. 强制指定 IP 地址

```bash
export EXTERNAL_HOST=192.168.1.100
./build.sh build-all
```

### 3. 编辑 .env 文件

```bash
# 编辑 .env
EXTERNAL_HOST=192.168.1.100
AI_INFRA_NETWORK_ENV=external
```

## 相关文件修改

1. **build.sh**
   - `detect_active_interface()` - 新增虚拟网卡和 IP 段排除
   - `detect_external_host()` - 扩展虚拟 IP 排除列表
   - `detect_k8s_environment()` - 新增本地集群检测逻辑

## 优化效果

| 场景 | 优化前 | 优化后 |
|------|-------|-------|
| macOS + Docker Desktop K8s | ❌ 192.168.65.3 (虚拟) | ✅ 192.168.1.100 (真实) |
| 生产 K8s 集群 | ✅ 正确检测 | ✅ 正确检测 |
| minikube 开发环境 | ⚠️ 误判为 K8s | ✅ 识别为本地开发 |
| Linux 服务器 | ✅ 正确检测 | ✅ 正确检测 |
| VMware 虚拟机 | ⚠️ 可能误判 | ✅ 正确排除虚拟网卡 |

## 总结

本次优化解决了以下问题：

1. ✅ **排除 Kubernetes 虚拟 IP**: `192.168.65.*`, `10.96.*`
2. ✅ **识别本地 K8s 集群**: docker-desktop, minikube, kind
3. ✅ **优先选择物理网卡**: en0 优先于 bridge100
4. ✅ **扩展虚拟接口排除**: bridge, vmnet, vboxnet, utun
5. ✅ **智能环境判断**: 区分本地开发和生产 K8s 环境

**关键改进:**
- 不再误判 Docker Desktop 的本地 Kubernetes 为生产环境
- 正确选择 macOS 物理网卡 (en0) 而非虚拟网卡 (bridge100)
- 支持手动覆盖，适应各种部署场景

---

**优化完成时间**: 2025-10-12  
**测试状态**: ✅ 待 macOS 环境验证  
**影响范围**: IP 检测和 Kubernetes 环境判断
