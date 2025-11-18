# SLURM节点Down状态修复指南

## 问题诊断

### 当前状态
```bash
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      3  down* test-ssh[01-03]

$ docker exec ai-infra-slurm-master sinfo -Nel
NODELIST    NODES PARTITION       STATE CPUS    S:C:T MEMORY TMP_DISK WEIGHT AVAIL_FE REASON              
test-ssh01      1  compute*       down* 2       1:2:1   1000        0      1   (null) Not responding      
test-ssh02      1  compute*       down* 2       1:2:1   1000        0      1   (null) Not responding      
test-ssh03      1  compute*       down* 2       1:2:1   1000        0      1   (null) Not responding
```

### 根本原因
1. ✅ 节点已在SLURM master配置中注册
2. ❌ 计算节点未安装`slurmd`守护进程
3. ❌ 导致SLURM master无法与节点通信

## 修复方案

### 方案1: 在实际物理/虚拟机上部署SLURM节点

#### 步骤1: 准备计算节点

在目标节点（test-ssh01, test-ssh02, test-ssh03）上执行以下操作：

```bash
# 1. 创建munge用户和组
sudo groupadd -g 1108 munge
sudo useradd -m -c "Munge Uid 'N' Gid Emporium" -d /var/lib/munge -u 1108 -g munge -s /sbin/nologin munge

# 2. 创建slurm用户和组
sudo groupadd -g 1109 slurm
sudo useradd -m -c "Slurm manager" -d /var/lib/slurm -u 1109 -g slurm -s /bin/bash slurm

# 3. 安装依赖包
sudo apt-get update
sudo apt-get install -y make hwloc libhwloc-dev libmunge-dev libmunge2 munge \
  liblua5.3-0 libfreeipmi17 libjwt0 libb64-0d libipmimonitoring6 \
  librdkafka1 freeipmi-common libmysqlclient-dev rng-tools

# 4. 配置rng-tools
sudo sed -i 's#^ExecStart=/usr/sbin/rngd -f#ExecStart=/sbin/rngd -f -r /dev/urandom#' /usr/lib/systemd/system/rngd.service
sudo systemctl daemon-reload
sudo systemctl enable rngd
sudo systemctl start rngd

# 5. 部署munge.key（从master复制）
sudo scp root@ai-infra-slurm-master:/etc/munge/munge.key /etc/munge/
sudo chown munge:munge /etc/munge/munge.key
sudo chmod 400 /etc/munge/munge.key

# 6. 启动munge服务
sudo systemctl enable munge
sudo systemctl start munge
sudo systemctl status munge

# 7. 测试munge认证
munge -n | ssh root@ai-infra-slurm-master unmunge
# 应该看到: STATUS:           Success (0)
```

#### 步骤2: 安装SLURM客户端和slurmd

```bash
# 1. 从AppHub或编译安装SLURM
# 假设使用AppHub的APK包
wget http://your-apphub-server/slurm/slurm-24.05.4-r0.apk
wget http://your-apphub-server/slurm/slurmd-24.05.4-r0.apk

sudo apk add --allow-untrusted slurm-24.05.4-r0.apk slurmd-24.05.4-r0.apk

# 2. 创建SLURM配置目录
sudo mkdir -p /etc/slurm /var/spool/slurmd /var/log/slurm

# 3. 从master复制slurm.conf
sudo scp root@ai-infra-slurm-master:/etc/slurm/slurm.conf /etc/slurm/

# 4. 设置权限
sudo chown -R slurm:slurm /var/spool/slurmd /var/log/slurm /etc/slurm

# 5. 创建slurmd systemd服务
sudo tee /etc/systemd/system/slurmd.service <<EOF
[Unit]
Description=Slurm node daemon
After=network.target munge.service
Requires=munge.service

[Service]
Type=forking
EnvironmentFile=-/etc/default/slurmd
ExecStart=/usr/sbin/slurmd $SLURMD_OPTIONS
ExecReload=/bin/kill -HUP \$MAINPID
PIDFile=/var/run/slurmd.pid
KillMode=process
LimitNOFILE=131072
LimitMEMLOCK=infinity
LimitSTACK=infinity
Delegate=yes

[Install]
WantedBy=multi-user.target
EOF

# 6. 启动slurmd
sudo systemctl daemon-reload
sudo systemctl enable slurmd
sudo systemctl start slurmd
sudo systemctl status slurmd
```

#### 步骤3: 验证节点状态

在SLURM master上执行：

```bash
# 查看节点状态
docker exec ai-infra-slurm-master sinfo -Nel

# 应该看到节点状态从 down* 变为 idle 或 idle~
# idle~  表示节点刚启动，等待被标记为可用
# idle   表示节点完全就绪

# 如果节点仍然是 down*，手动恢复：
docker exec ai-infra-slurm-master scontrol update NodeName=test-ssh01 State=RESUME
docker exec ai-infra-slurm-master scontrol update NodeName=test-ssh02 State=RESUME
docker exec ai-infra-slurm-master scontrol update NodeName=test-ssh03 State=RESUME

# 再次检查
docker exec ai-infra-slurm-master sinfo
```

### 方案2: 使用Docker容器模拟计算节点

如果要在Docker环境中测试，需要创建slurmd容器：

#### 创建Slurmd Dockerfile

```dockerfile
# src/slurm-node/Dockerfile
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# 安装基础依赖
RUN apt-get update && apt-get install -y \
    munge libmunge2 libmunge-dev \
    hwloc libhwloc-dev \
    libmysqlclient-dev \
    openssh-server \
    supervisor \
    && rm -rf /var/lib/apt/lists/*

# 创建用户
RUN groupadd -g 1108 munge && \
    useradd -m -c "Munge Uid 'N' Gid Emporium" -d /var/lib/munge -u 1108 -g munge -s /sbin/nologin munge && \
    groupadd -g 1109 slurm && \
    useradd -m -c "Slurm manager" -d /var/lib/slurm -u 1109 -g slurm -s /bin/bash slurm

# 复制SLURM二进制文件（从构建镜像或AppHub）
COPY --from=ai-infra-slurm-master:latest /usr/bin/slurmd /usr/sbin/
COPY --from=ai-infra-slurm-master:latest /usr/lib/*slurm* /usr/lib/

# 创建目录
RUN mkdir -p /etc/slurm /var/spool/slurmd /var/log/slurm && \
    chown -R slurm:slurm /var/spool/slurmd /var/log/slurm

# 复制配置文件（在docker-compose中挂载）
VOLUME ["/etc/slurm", "/etc/munge", "/var/spool/slurmd"]

# 启动脚本
COPY start-slurmd.sh /start-slurmd.sh
RUN chmod +x /start-slurmd.sh

CMD ["/start-slurmd.sh"]
```

#### 启动脚本

```bash
#!/bin/bash
# src/slurm-node/start-slurmd.sh

# 启动munge
munged

# 等待munge就绪
sleep 2

# 启动slurmd
exec /usr/sbin/slurmd -D
```

#### Docker Compose配置

```yaml
# 在docker-compose.yml中添加
services:
  slurm-node-01:
    image: ai-infra-slurm-node:latest
    build:
      context: ./src/slurm-node
      dockerfile: Dockerfile
    container_name: test-ssh01
    hostname: test-ssh01
    networks:
      - ai-infra-network
    volumes:
      - ./data/slurm/node01/spool:/var/spool/slurmd
      - slurm-config:/etc/slurm:ro
      - munge-key:/etc/munge:ro
    depends_on:
      - slurm-master
      
  slurm-node-02:
    image: ai-infra-slurm-node:latest
    container_name: test-ssh02
    hostname: test-ssh02
    networks:
      - ai-infra-network
    volumes:
      - ./data/slurm/node02/spool:/var/spool/slurmd
      - slurm-config:/etc/slurm:ro
      - munge-key:/etc/munge:ro
    depends_on:
      - slurm-master
      
  slurm-node-03:
    image: ai-infra-slurm-node:latest
    container_name: test-ssh03
    hostname: test-ssh03
    networks:
      - ai-infra-network
    volumes:
      - ./data/slurm/node03/spool:/var/spool/slurmd
      - slurm-config:/etc/slurm:ro
      - munge-key:/etc/munge:ro
    depends_on:
      - slurm-master

volumes:
  slurm-config:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /var/run/ai-infra/slurm/config
      
  munge-key:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /var/run/ai-infra/munge
```

## 使用Backend API管理节点

### API端点

```bash
# 1. 执行SLURM命令
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -X POST http://192.168.0.200:8080/api/slurm/exec \
  -d '{"command":"sinfo"}'

# 2. 获取诊断信息
curl -H "Authorization: Bearer $TOKEN" \
  http://192.168.0.200:8080/api/slurm/diagnostics

# 3. 恢复down节点
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -X POST http://192.168.0.200:8080/api/slurm/exec \
  -d '{"command":"scontrol update NodeName=test-ssh01 State=RESUME"}'
```

### 允许的SLURM命令

Backend API只允许执行以下安全的SLURM命令：
- `sinfo` - 查看节点信息
- `squeue` - 查看作业队列
- `scontrol` - 控制命令
- `sacct` - 作业统计
- `sstat` - 作业状态
- `srun` - 运行作业
- `sbatch` - 批量提交作业
- `scancel` - 取消作业

## SLURM REST API部署

### 在SLURM Master上安装slurmrestd

```bash
# 1. 进入master容器
docker exec -it ai-infra-slurm-master bash

# 2. 安装slurmrestd（如果未安装）
apt-get update
apt-get install -y slurm-wlm-rest-api

# 3. 创建JWT认证密钥
mkdir -p /var/spool/slurm/statesave
dd if=/dev/random of=/var/spool/slurm/statesave/jwt_hs256.key bs=32 count=1
chown slurm:slurm /var/spool/slurm/statesave/jwt_hs256.key
chmod 600 /var/spool/slurm/statesave/jwt_hs256.key

# 4. 配置slurm.conf添加AuthAltTypes
echo "AuthAltTypes=auth/jwt" >> /etc/slurm/slurm.conf
scontrol reconfigure

# 5. 创建systemd服务
tee /etc/systemd/system/slurmrestd.service <<EOF
[Unit]
Description=Slurm REST API daemon
After=network.target munge.service slurmctld.service
Requires=munge.service

[Service]
Type=simple
Environment="SLURM_JWT=daemon"
ExecStart=/usr/sbin/slurmrestd 0.0.0.0:6820 -vvv
User=slurm
Group=slurm
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

# 6. 启动slurmrestd
systemctl daemon-reload
systemctl enable slurmrestd
systemctl start slurmrestd
systemctl status slurmrestd
```

### 暴露REST API端口

在`docker-compose.yml`中添加端口映射：

```yaml
services:
  slurm-master:
    ports:
      - "6817:6817"  # slurmctld
      - "6818:6818"  # slurmdbd
      - "6820:6820"  # slurmrestd (新增)
      - "22:22"      # SSH
```

### 测试REST API

```bash
# 1. 生成JWT Token
TOKEN=$(docker exec ai-infra-slurm-master scontrol token username=slurm)

# 2. 测试API
curl -H "X-SLURM-USER-NAME:slurm" \
  -H "X-SLURM-USER-TOKEN:$TOKEN" \
  http://192.168.0.200:6820/slurm/v0.0.40/diag

# 3. 获取节点信息
curl -H "X-SLURM-USER-NAME:slurm" \
  -H "X-SLURM-USER-TOKEN:$TOKEN" \
  http://192.168.0.200:6820/slurm/v0.0.40/nodes

# 4. 获取作业信息
curl -H "X-SLURM-USER-NAME:slurm" \
  -H "X-SLURM-USER-TOKEN:$TOKEN" \
  http://192.168.0.200:6820/slurm/v0.0.40/jobs
```

## 验证修复

### 检查清单

- [ ] munge服务在所有节点上运行
- [ ] munge认证测试成功
- [ ] slurmd在计算节点上运行
- [ ] 节点状态从down*变为idle
- [ ] SLURM REST API可访问
- [ ] Backend API可以执行SLURM命令

### 最终验证

```bash
# 1. 检查节点状态
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      3   idle test-ssh[01-03]

# 2. 提交测试作业
$ docker exec ai-infra-slurm-master srun -N1 hostname
test-ssh01

# 3. 检查作业历史
$ docker exec ai-infra-slurm-master sacct
JobID    JobName  Partition    Account  AllocCPUS      State ExitCode 
1        hostname  compute        root          2  COMPLETED      0:0
```

## 参考Ansible Playbook

参考文档提供的Ansible配置可以自动化部署计算节点：

```yaml
---
- name: 部署SLURM计算节点
  hosts: "{{ target_hosts }}"
  become: yes
  gather_facts: true
  vars:
    master_node: ai-infra-slurm-master
    package_list:
      - make
      - hwloc
      - libhwloc-dev
      - libmunge-dev
      - libmunge2
      - munge
      - liblua5.3-0
      # ... 其他包
      
  tasks:
    - name: 创建munge用户和组
      # ... (参考提供的playbook)
      
    - name: 创建slurm用户和组
      # ... 
      
    - name: 安装依赖包
      # ...
      
    - name: 部署munge.key
      # ...
      
    - name: 启动munge服务
      # ...
      
    # 添加slurmd安装和配置任务
```

## 总结

修复SLURM节点down*状态需要：

1. 在计算节点上安装并配置slurmd
2. 确保munge认证正常工作
3. 配置SLURM REST API（可选但推荐）
4. 使用Backend API进行运维管理

建议使用Ansible自动化部署，确保配置一致性。
