# AI Infrastructure Matrix 用户操作手册

## 概述

本手册提供 AI Infrastructure Matrix 平台的详细使用说明，帮助用户快速上手各项功能。

## 目录

- [登录与认证](#登录与认证)
- [JupyterHub 使用](#jupyterhub-使用)
- [Gitea 代码仓库](#gitea-代码仓库)
- [Slurm 作业管理](#slurm-作业管理)
- [对象存储](#对象存储)
- [监控面板](#监控面板)

## 登录与认证

### 首次登录

1. 打开浏览器访问 `http://localhost:8080`
2. 使用默认管理员账号登录：
   - 用户名：`admin`
   - 密码：`admin123`
3. 首次登录后建议立即修改密码

### 用户管理

管理员可以在后台管理界面创建和管理用户账号。

## JupyterHub 使用

### 访问 JupyterHub

访问 `http://localhost:8080/jupyter` 进入 JupyterHub 环境。

### 创建 Notebook

1. 登录后点击 "New" 按钮
2. 选择 Python 3 内核
3. 开始编写代码

### GPU 资源使用

如果配置了 GPU 资源，可以在 Notebook 中使用：

```python
import torch
print(torch.cuda.is_available())
```

更多详情参考：[JupyterHub使用指南](JUPYTERHUB_UNIFIED_AUTH_GUIDE.md)

## Gitea 代码仓库

### 访问 Gitea

访问 `http://localhost:8080/gitea/` 进入 Gitea 代码仓库。

### 创建仓库

1. 登录后点击右上角 "+" 按钮
2. 选择 "新建仓库"
3. 填写仓库名称和描述
4. 选择公开或私有
5. 点击 "创建仓库"

### 克隆仓库

```bash
git clone http://localhost:8080/gitea/username/repository.git
```

### LFS 大文件存储

Gitea 已配置 MinIO 作为 LFS 后端，支持大文件存储：

```bash
# 安装 Git LFS
git lfs install

# 跟踪大文件
git lfs track "*.psd"
git add .gitattributes
git commit -m "Track PSD files"
```

## Slurm 作业管理

### 访问 Slurm 管理界面

在主界面导航到 "Slurm 集群管理"。

### 提交作业

1. 点击 "作业管理" -> "新建作业"
2. 填写作业参数：
   - 作业名称
   - 队列（分区）
   - 节点数
   - CPU/内存需求
3. 上传或编写作业脚本
4. 点击 "提交"

### 查看作业状态

在作业列表中可以查看：
- 作业ID
- 状态（排队/运行/完成/失败）
- 运行时间
- 资源使用情况

### 节点管理

管理员可以：
- 添加计算节点
- 查看节点状态
- 配置分区（队列）
- 设置资源限制

## 对象存储

### 访问 MinIO 控制台

访问 `http://localhost:8080/minio-console/`

### 创建存储桶

1. 登录控制台
2. 点击 "Buckets" -> "Create Bucket"
3. 输入桶名称
4. 配置访问策略
5. 点击 "Create"

### 上传文件

1. 选择存储桶
2. 点击 "Upload" 按钮
3. 选择文件或拖拽上传

### S3 API 访问

使用 AWS CLI 或 boto3 访问：

```python
import boto3

s3 = boto3.client('s3',
    endpoint_url='http://localhost:8080/minio',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minioadmin'
)

# 列出所有桶
buckets = s3.list_buckets()
```

## 监控面板

### 访问 Nightingale

访问 `http://localhost:8080/n9e` 进入监控系统。

### 查看仪表盘

- 系统概览
- 节点监控
- 服务状态
- 资源使用趋势

### 配置告警

1. 进入 "告警规则"
2. 点击 "新建规则"
3. 配置监控指标和阈值
4. 设置通知方式
5. 保存规则

## 常见问题

### 忘记密码

联系管理员重置密码。

### 作业一直排队

检查：
1. 集群节点是否在线
2. 资源配额是否充足
3. 队列配置是否正确

### 服务访问失败

1. 检查服务状态：`docker compose ps`
2. 查看服务日志：`docker compose logs [服务名]`
3. 确认网络连接正常

## 获取帮助

- 📧 技术支持邮箱：support@example.com
- 📖 详细文档：[docs/](.)
- 🐛 问题反馈：[GitHub Issues](https://github.com/aresnasa/ai-infra-matrix/issues)
