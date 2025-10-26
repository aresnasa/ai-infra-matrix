# 网络代理容器

## 功能说明

该代理容器使用HAProxy将Docker内部网络中的数据库和缓存服务代理到宿主机网络，方便开发和调试。

## 支持的服务

- **PostgreSQL**: 端口 5432
- **MySQL**: 端口 3306
- **Redis**: 端口 6379
- **OceanBase**: 端口 2881

## 构建镜像

```bash
docker build -t ai-infra-proxy:latest -f src/proxy/Dockerfile src/proxy
```

## 使用方法

### 1. 在docker-compose.yml中添加服务

```yaml
  proxy:
    build:
      context: ./src/proxy
      dockerfile: Dockerfile
    container_name: ai-infra-proxy
    ports:
      - "5432:5432"   # PostgreSQL
      - "3306:3306"   # MySQL
      - "6379:6379"   # Redis
      - "2881:2881"   # OceanBase
      - "8404:8404"   # HAProxy统计页面
    networks:
      - ai-infra-network
    depends_on:
      - postgres
      - mysql
      - redis
      - oceanbase
    restart: unless-stopped
```

### 2. 启动服务

```bash
docker-compose up -d proxy
```

### 3. 从宿主机连接

连接到代理后的服务：

```bash
# PostgreSQL
psql -h localhost -p 5432 -U postgres -d ai-infra-matrix

# MySQL
mysql -h localhost -P 3306 -u root -p

# Redis
redis-cli -h localhost -p 6379

# OceanBase (使用obclient或MySQL客户端)
mysql -h localhost -P 2881 -u root
```

## 监控和管理

访问HAProxy统计页面查看代理状态：

```
http://localhost:8404/stats
```

默认用户名/密码: `admin/admin`

## 配置文件

- `Dockerfile`: 容器构建文件
- `haproxy.cfg`: HAProxy配置文件

## 自定义配置

如需修改代理规则，编辑 `haproxy.cfg` 文件后重新构建镜像：

```bash
docker-compose build proxy
docker-compose up -d proxy
```

## 注意事项

1. 该代理容器主要用于开发环境，生产环境建议使用更安全的网络配置
2. 确保宿主机的相应端口未被占用
3. 代理服务需要与被代理的服务在同一Docker网络中
4. 健康检查会定期检查HAProxy配置的有效性

## 故障排除

### 连接失败

检查容器日志：
```bash
docker logs ai-infra-proxy
```

### 端口冲突

如果端口已被占用，修改docker-compose.yml中的端口映射：
```yaml
ports:
  - "15432:5432"  # 使用其他端口
```

### 查看HAProxy状态

```bash
docker exec ai-infra-proxy haproxy -c -f /usr/local/etc/haproxy/haproxy.cfg
```
