services:
  # PostgreSQL 数据库服务（主要用于门户与Gitea等组件）
  postgres:
    image: postgres:15-alpine
    container_name: ai-infra-postgres
    env_file:
      - .env
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-ai-infra-matrix}
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
      TZ: Asia/Shanghai
    expose:
      - "5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-ai-infra-matrix}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped

  # OceanBase 社区版（单机）
  # 参考: https://www.oceanbase.com/docs/community-observer-cn-10000000001878977
  oceanbase:
    image: oceanbase/oceanbase-ce:4.3.5-lts
    container_name: ai-infra-oceanbase
    environment:
      TZ: Asia/Shanghai
    # 对外映射 2881 端口，供客户端连接（obclient, JDBC等）
    ports:
      - "2881:2881"
    expose:
      - "2881"
    # 初始化SQL脚本目录，容器启动时会自动执行 /root/boot/init.d 下的 *.sql / *.sh
    # 可通过 OCEANBASE_INIT_DIR 指定宿主机路径（默认 ./data/oceanbase/init.d）
    volumes:
      - ${OCEANBASE_INIT_DIR:-./data/oceanbase/init.d}:/root/boot/init.d:ro
    networks:
      - ai-infra-network
    healthcheck:
      # OceanBase 健康检查：分阶段检查
      # 1. 检查 observer 进程是否运行
      # 2. 检查端口是否监听
      # 3. 尝试数据库连接（可选，因为初始化可能较慢）
      test:
        - "CMD-SHELL"
        - |
          # 检查 observer 进程
          if ! pgrep -f observer >/dev/null 2>&1; then
            echo "OceanBase observer process not running" && exit 1
          fi
          # 检查端口监听（使用 ss 或 netstat 替代 nc）
          if command -v ss >/dev/null 2>&1; then
            ss -tln | grep :2881 >/dev/null 2>&1 || (echo "Port 2881 not listening" && exit 1)
          elif command -v netstat >/dev/null 2>&1; then
            netstat -tln | grep :2881 >/dev/null 2>&1 || (echo "Port 2881 not listening" && exit 1)
          else
            echo "Port check skipped (no ss/netstat available)"
          fi
          # 如果一切正常，返回成功
          echo "OceanBase health check passed"
      interval: 30s
      timeout: 15s
      retries: 10
      start_period: 120s
    restart: unless-stopped

  # MySQL 数据库服务 (用于SLURM)
  mysql:
    image: mysql:8.0
    container_name: ai-infra-mysql
    environment:
      TZ: Asia/Shanghai
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-mysql123}
      MYSQL_DATABASE: ${SLURM_DB_NAME:-slurm_acct_db}
      MYSQL_USER: ${SLURM_DB_USER:-slurm}
      MYSQL_PASSWORD: ${SLURM_DB_PASSWORD:-slurm123}
    expose:
      - "3306"
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD:-mysql123}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped

# Redis 缓存服务
  redis:
    image: redis:7-alpine
    container_name: ai-infra-redis
    env_file:
      - .env
    # Defer expansion of REDIS_PASSWORD to the container environment
    command: ["sh", "-c", "exec redis-server --requirepass \"$${REDIS_PASSWORD}\""]
    expose:
      - "6379"
    volumes:
      - redis_data:/data
    networks:
      - ai-infra-network
    environment:
      TZ: Asia/Shanghai
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a \"$${REDIS_PASSWORD}\" ping || exit 1"]
      interval: 10s
      timeout: 3s
      retries: 5
      start_period: 10s
    restart: unless-stopped

# Kafka 消息队列服务 (KRaft模式，无需Zookeeper)
  kafka:
    image: confluentinc/cp-kafka:7.5.0
    container_name: ai-infra-kafka
    environment:
      # KRaft 模式配置
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@kafka:9093
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:9094
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093,PLAINTEXT_HOST://0.0.0.0:9094
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      
      # 集群和主题配置
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: 'true'
      KAFKA_NUM_PARTITIONS: 3
      KAFKA_DEFAULT_REPLICATION_FACTOR: 1
      
      # 日志和存储配置
      KAFKA_LOG_RETENTION_HOURS: 168  # 7天
      KAFKA_LOG_SEGMENT_BYTES: 1073741824  # 1GB
      KAFKA_LOG_RETENTION_BYTES: 10737418240  # 10GB
      KAFKA_LOG_DIRS: /var/lib/kafka/data
      
      # 性能优化
      KAFKA_SOCKET_SEND_BUFFER_BYTES: 102400
      KAFKA_SOCKET_RECEIVE_BUFFER_BYTES: 102400
      KAFKA_SOCKET_REQUEST_MAX_BYTES: 104857600
      KAFKA_NUM_NETWORK_THREADS: 8
      KAFKA_NUM_IO_THREADS: 8
      KAFKA_BACKGROUND_THREADS: 10
      
      # 集群元数据配置（KRaft）
      CLUSTER_ID: 'gYf__u4_TgSoREBUnP-YzQ'
      
      TZ: Asia/Shanghai
    expose:
      - "9092"
      - "9093"
    ports:
      - "9094:9094"  # 外部访问端口
    volumes:
      - kafka_data:/var/lib/kafka/data
      - kafka_logs:/var/log/kafka
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "kafka-broker-api-versions", "--bootstrap-server", "localhost:9092"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
    restart: unless-stopped

# Kafka UI 管理界面
  kafka-ui:
    image: provectuslabs/kafka-ui:latest
    container_name: ai-infra-kafka-ui
    depends_on:
      kafka:
        condition: service_healthy
    volumes:
      - ./config/kafka-ui-config.yml:/etc/kafka-ui/dynamic_config.yml
    environment:
      DYNAMIC_CONFIG_ENABLED: 'true'
      TZ: Asia/Shanghai
      # 设置基础路径以支持反向代理
      SERVER_SERVLET_CONTEXT_PATH: /kafka-ui-backend
    expose:
      - "8080"
    ports:
      - "9095:8080"  # Kafka UI 管理界面
    networks:
      - ai-infra-network
    restart: unless-stopped

  # OpenLDAP 目录服务
  openldap:
    image: osixia/openldap:stable
    container_name: ai-infra-openldap
    env_file:
      - .env
    environment:
      LDAP_ORGANISATION: "${LDAP_ORGANISATION:-AI Infrastructure}"
      LDAP_DOMAIN: "${LDAP_DOMAIN:-ai-infra.com}"
      LDAP_ADMIN_PASSWORD: "${LDAP_ADMIN_PASSWORD}"
      LDAP_CONFIG_PASSWORD: "${LDAP_CONFIG_PASSWORD}"
      LDAP_BASE_DN: "${LDAP_BASE_DN:-dc=ai-infra,dc=com}"
      TZ: "${TZ:-Asia/Shanghai}"
    expose:
      - "389"
      - "636"
    volumes:
      - ldap_data:/var/lib/ldap
      - ldap_config:/etc/ldap/slapd.d
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "sh", "-c", "ldapsearch -x -H ldap://localhost -b dc=ai-infra,dc=com -D cn=admin,dc=ai-infra,dc=com -w \"$$LDAP_ADMIN_PASSWORD\""]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
    restart: unless-stopped

  # phpLDAPadmin Web界面
  phpldapadmin:
    image: osixia/phpldapadmin:stable
    container_name: ai-infra-phpldapadmin
    environment:
      PHPLDAPADMIN_LDAP_HOSTS: "${LDAP_HOST:-openldap}"
      PHPLDAPADMIN_HTTPS: "${PHPLDAPADMIN_HTTPS:-false}"
      TZ: "${TZ:-Asia/Shanghai}"
    expose:
      - "80"
    depends_on:
      openldap:
        condition: service_healthy
    networks:
      - ai-infra-network
    restart: unless-stopped

  # 后端初始化服务 - 创建admin用户和基础数据
  backend-init:
    image: ai-infra-backend-init:{{IMAGE_TAG}}
    build:
      context: ./src/backend
      dockerfile: Dockerfile
      target: backend-init
      args:
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-backend-init
    env_file:
      - .env
    environment:
      # 默认仍支持 Postgres（未启用 OceanBase 时）
      - DB_HOST=${POSTGRES_HOST:-postgres}
      - DB_PORT=${POSTGRES_PORT:-5432}
      - DB_USER=${POSTGRES_USER:-postgres}
      - DB_PASSWORD=${POSTGRES_PASSWORD:-postgres}
      - DB_NAME=${POSTGRES_DB:-ai-infra-matrix}
      # OceanBase（可选）
      - OB_ENABLED=${OB_ENABLED:-false}
      - OB_HOST=${OB_HOST:-oceanbase}
      - OB_PORT=${OB_PORT:-2881}
      - OB_USER=${OB_USER:-root@sys}
      - OB_PASSWORD=${OB_PASSWORD:-}
      - OB_DB=${OB_DB:-aimatrix}
      - OB_PARAMS=${OB_PARAMS:-charset=utf8mb4&parseTime=True&loc=Local}
      - REDIS_HOST=${REDIS_HOST:-redis}
      - REDIS_PORT=${REDIS_PORT:-6379}
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - LDAP_SERVER=${LDAP_HOST:-openldap}
      - LDAP_PORT=${LDAP_PORT:-389}
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - TZ=Asia/Shanghai
    command: ["./init"]
    depends_on:
      oceanbase:
        condition: service_healthy
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      openldap:
        condition: service_healthy
    networks:
      - ai-infra-network
    restart: "no"

  # 后端 API 服务
  backend:
    image: ai-infra-backend:{{IMAGE_TAG}}
    build:
      context: ./src/backend
      dockerfile: Dockerfile
      target: backend
      args:
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-backend
    env_file:
      - .env
    environment:
      # 默认仍支持 Postgres
      DB_HOST: "${POSTGRES_HOST:-postgres}"
      DB_PORT: "${POSTGRES_PORT:-5432}"
      DB_USER: "${POSTGRES_USER:-postgres}"
      DB_PASSWORD: "${POSTGRES_PASSWORD:-postgres}"
      DB_NAME: "${POSTGRES_DB:-ai-infra-matrix}"
      # 开启 OceanBase 将覆盖上面的连接，后端将优先使用 OceanBase
      OB_ENABLED: "${OB_ENABLED:-false}"
      OB_HOST: "${OB_HOST:-oceanbase}"
      OB_PORT: "${OB_PORT:-2881}"
      OB_USER: "${OB_USER:-root@sys}"
      OB_PASSWORD: "${OB_PASSWORD:-}"
      OB_DB: "${OB_DB:-aimatrix}"
      OB_PARAMS: "${OB_PARAMS:-charset=utf8mb4&parseTime=True&loc=Local}"
      REDIS_HOST: "${REDIS_HOST:-redis}"
      REDIS_PORT: "${REDIS_PORT:-6379}"
      REDIS_PASSWORD: "${REDIS_PASSWORD}"
      LDAP_SERVER: "${LDAP_HOST:-openldap}"
      LDAP_PORT: "${LDAP_PORT:-389}"
      LDAP_BASE_DN: "${LDAP_BASE_DN:-dc=example,dc=org}"
      # 外部访问地址（用于外部节点连接 Salt Master 等场景）
      EXTERNAL_HOST: "${EXTERNAL_HOST:-}"
      # SLURM Master SSH 配置（用于缩容等操作）
      SLURM_MASTER_HOST: "${SLURM_MASTER_HOST:-slurm-master}"
      SLURM_MASTER_PORT: "${SLURM_MASTER_PORT:-22}"
      SLURM_MASTER_USER: "${SLURM_MASTER_USER:-root}"
      SLURM_MASTER_PASSWORD: "${SLURM_MASTER_PASSWORD}"
      # SaltStack 配置 (多 Master 高可用)
      SALTSTACK_ENABLED: "${SALTSTACK_ENABLED:-true}"
      SALTSTACK_MASTER_HOST: "${SALTSTACK_MASTER_HOST:-${EXTERNAL_HOST}}"
      SALTSTACK_MASTER_URL: "${SALTSTACK_MASTER_URL:-}"
      # 多 Master URL 列表 (逗号分隔，用于高可用故障转移)
      SALT_MASTER_URLS: "${SALT_MASTER_URLS:-http://salt-master-1:8002,http://salt-master-2:8002}"
      SALT_API_SCHEME: "${SALT_API_SCHEME:-http}"
      SALT_MASTER_HOST: "${SALT_MASTER_HOST:-salt-master-1}"
      SALT_API_PORT: "${SALT_API_PORT:-8002}"
      SALT_API_USERNAME: "${SALT_API_USERNAME:-saltapi}"
      SALT_API_PASSWORD: "${SALT_API_PASSWORD:-aiinfra-salt}"
      SALT_API_EAUTH: "${SALT_API_EAUTH:-file}"
      SALT_API_TIMEOUT: "${SALT_API_TIMEOUT:-65s}"
      GITEA_ENABLED: "${GITEA_ENABLED:-true}"
      GITEA_BASE_URL: "${GITEA_BASE_URL:-http://gitea:3000}"
      GITEA_ADMIN_TOKEN: "${GITEA_ADMIN_TOKEN}"
      GITEA_AUTO_CREATE: "${GITEA_AUTO_CREATE:-true}"
      GITEA_AUTO_UPDATE: "${GITEA_AUTO_UPDATE:-true}"
      GITEA_SYNC_ENABLED: "${GITEA_SYNC_ENABLED:-true}"
      GITEA_SYNC_INTERVAL_SECONDS: "${GITEA_SYNC_INTERVAL_SECONDS:-600}"
      # Map reserved username 'admin' to a real admin account in Gitea (use 'admin' as default)
      GITEA_ALIAS_ADMIN_TO: "${GITEA_ALIAS_ADMIN_TO:-admin}"
      # SeaweedFS 对象存储配置
      SEAWEEDFS_FILER_HOST: "${SEAWEEDFS_FILER_HOST:-seaweedfs-filer}"
      SEAWEEDFS_FILER_PORT: "${SEAWEEDFS_FILER_PORT:-8888}"
      SEAWEEDFS_MASTER_HOST: "${SEAWEEDFS_MASTER_HOST:-seaweedfs-master}"
      SEAWEEDFS_MASTER_PORT: "${SEAWEEDFS_MASTER_PORT:-9333}"
      SEAWEEDFS_S3_PORT: "${SEAWEEDFS_S3_PORT:-8333}"
      SEAWEEDFS_ACCESS_KEY: "${SEAWEEDFS_ACCESS_KEY:-seaweedfs_admin}"
      SEAWEEDFS_SECRET_KEY: "${SEAWEEDFS_SECRET_KEY:-seaweedfs_secret_key_change_me}"
      SEAWEEDFS_USE_SSL: "${SEAWEEDFS_USE_SSL:-false}"
      # E2E 测试配置（开发/测试环境）
      E2E_ALLOW_FAKE_LDAP: "${E2E_ALLOW_FAKE_LDAP:-true}"
      # 注册安全策略
      REGISTRATION_REQUIRE_INVITATION_CODE: "${REGISTRATION_REQUIRE_INVITATION_CODE:-true}"
      LOG_LEVEL: "${LOG_LEVEL:-info}"
      TZ: "Asia/Shanghai"
      # 外部脚本目录 - 优先加载外部脚本模板
      SCRIPTS_DIR: /app/scripts
    expose:
      - "8082"
    ports:
      # Debug: expose internal 8082 on host for direct backend access (bypass Nginx)
      - "${BIND_HOST:-0.0.0.0}:${BACKEND_DEBUG_PORT:-8082}:8082"
    extra_hosts:
      - "kubernetes.docker.internal:host-gateway"
      - "host.docker.internal:host-gateway"
    depends_on:
      oceanbase:
        condition: service_healthy
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      openldap:
        condition: service_healthy
      backend-init:
        condition: service_completed_successfully
      seaweedfs-filer:
        condition: service_healthy
    volumes:
      - ./src/backend/outputs:/app/outputs
      - ./src/backend/uploads:/app/uploads
      # 外部脚本目录 - 优先加载，可覆盖内置脚本模板
      - ./src/backend/internal/services/scripts:/app/scripts:ro
      # Docker socket - 用于获取 Salt Master 容器的监控指标
      - /var/run/docker.sock:/var/run/docker.sock:ro
      # Salt Master PKI 密钥 - 用于分发 master.pub 给新安装的 minion
      - salt_keys:/etc/salt/pki:ro
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/api/health"]
      interval: 30s
      timeout: 15s
      retries: 5
      start_period: 60s
    restart: unless-stopped

  # 前端应用服务
  frontend:
    image: ai-infra-frontend:{{IMAGE_TAG}}
    build:
      context: ./src/frontend
      dockerfile: Dockerfile
      args:
        REACT_APP_API_URL: /api
        REACT_APP_JUPYTERHUB_URL: /jupyter
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-frontend
    environment:
      TZ: Asia/Shanghai
    expose:
      - "80"
    depends_on:
      backend:
        condition: service_healthy
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    restart: unless-stopped

  # JupyterHub 统一认证服务
  jupyterhub:
    image: ai-infra-jupyterhub:{{IMAGE_TAG}}
    env_file:
      - .env
    build:
      context: ./src/jupyterhub
      dockerfile: Dockerfile
      args:
        BUILDKIT_INLINE_CACHE: "${BUILDKIT_INLINE_CACHE:-1}"
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-jupyterhub
    environment:
      - POSTGRES_HOST=${POSTGRES_HOST:-postgres}
      - POSTGRES_PORT=${POSTGRES_PORT:-5432}
      - POSTGRES_DB=jupyterhub_db
      - POSTGRES_USER=${POSTGRES_USER:-postgres}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-postgres}
      - DB_HOST=${POSTGRES_HOST:-postgres}
      - DB_PORT=${POSTGRES_PORT:-5432}
      - DB_NAME=jupyterhub_db
      - DB_USER=${POSTGRES_USER:-postgres}
      - DB_PASSWORD=${POSTGRES_PASSWORD:-postgres}
      - REDIS_HOST=${REDIS_HOST:-redis}
      - REDIS_PORT=${REDIS_PORT:-6379}
      - REDIS_DB=1
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - JWT_SECRET=${JWT_SECRET}
      - ENCRYPTION_KEY=${ENCRYPTION_KEY:-your-encryption-key-change-in-production-32-bytes}
      - JUPYTERHUB_ADMIN_USERS=admin,jupyter-admin
      # SSL/TLS 和外部访问配置
      - ENABLE_TLS=${ENABLE_TLS:-false}
      - EXTERNAL_SCHEME=${EXTERNAL_SCHEME:-http}
      - EXTERNAL_HOST=${EXTERNAL_HOST:-localhost}
      - EXTERNAL_PORT=${EXTERNAL_PORT:-8080}
      - HTTPS_PORT=${HTTPS_PORT:-8443}
      # 公有云环境：PUBLIC_HOST 用于浏览器重定向（域名或公网 IP）
      - PUBLIC_HOST=${PUBLIC_HOST:-}
      - CONFIGPROXY_AUTH_TOKEN=${CONFIGPROXY_AUTH_TOKEN}
      - JUPYTERHUB_CRYPT_KEY=${JUPYTERHUB_CRYPT_KEY:-a3d7c9e5b1f2048c7d9e3b6a5c1f08e2a7b3c9d5e1f2048c7d9e3b6a5c1f08e2}
      - SESSION_TIMEOUT=86400
      - USE_CUSTOM_AUTH=true
      - AI_INFRA_BACKEND_URL=${BACKEND_URL:-http://backend:8082}
      - AI_INFRA_API_TOKEN=ai-infra-hub-token
      - JUPYTERHUB_AUTO_LOGIN=true
      - JUPYTERHUB_DEV_MODE=true
      - JUPYTERHUB_IMAGE=ai-infra-singleuser:{{IMAGE_TAG}}
      - JUPYTERHUB_NETWORK=ai-infra-network
      - JUPYTERHUB_MEM_LIMIT=3G
      - JUPYTERHUB_CPU_LIMIT=1.0
      - JUPYTERHUB_IDLE_CULLER_ENABLED=true
      - JUPYTERHUB_IDLE_TIMEOUT=3600
      - JUPYTERHUB_CULL_TIMEOUT=7200
      - JUPYTERHUB_DEBUG=false
      - JUPYTERHUB_LOG_LEVEL=INFO
      - JUPYTERHUB_ACCESS_LOG=true
      - JUPYTERHUB_USE_PROXY=true
      - JUPYTERHUB_PUBLIC_HOST=${JUPYTERHUB_PUBLIC_HOST}
      - JUPYTERHUB_CORS_ORIGIN=${JUPYTERHUB_CORS_ORIGIN}
      - TZ=Asia/Shanghai
    ports:
      - "${BIND_HOST:-0.0.0.0}:${JUPYTERHUB_EXTERNAL_PORT}:8000"
    expose:
      - "8000"
      - "8091"
    volumes:
      - jupyterhub_data:/srv/data/jupyterhub
      - jupyterhub_notebooks:/srv/jupyterhub/notebooks
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./shared:/srv/jupyterhub/shared:rw
      - ./src/jupyterhub/jupyterhub_config.py:/srv/jupyterhub/jupyterhub_config.py:ro
      - ./src/jupyterhub/backend_integrated_config.py:/srv/jupyterhub/backend_integrated_config.py:ro
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      backend:
        condition: service_healthy
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/jupyter/hub/api"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
    restart: unless-stopped

  # 单用户镜像构建器（不运行，仅用于构建镜像供Spawner使用）
  singleuser-builder:
    image: ai-infra-singleuser:{{IMAGE_TAG}}
    build:
      context: ./src/singleuser
      dockerfile: Dockerfile
      args:
        VERSION: {{IMAGE_TAG}}
    command: ["true"]
    networks:
      - ai-infra-network
    restart: "no"

# ============================================================================
# SaltStack 多 Master 高可用架构
# ============================================================================
# 架构说明：
# - salt-master-1 (主节点): 负责生成和管理 PKI 密钥，对外暴露端口
# - salt-master-2 (备用节点): 共享同一套 PKI 密钥，提供故障转移
# - 所有 Master 共享 salt_keys volume，确保密钥一致
# - 后端通过 SaltMasterPool 自动选择健康的 Master
# ============================================================================

  # SaltStack Master 1 (主节点 - 负责端口暴露)
  salt-master-1:
    image: ai-infra-saltstack:{{IMAGE_TAG}}
    build:
      context: .
      dockerfile: ./src/saltstack/Dockerfile
      args:
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-salt-master-1
    hostname: salt-master-1
    privileged: true
    security_opt:
      - seccomp:unconfined
    # cgroup 配置: 支持 cgroupv1 和 cgroupv2
    # 自动检测: CGROUP_VERSION={{CGROUP_VERSION}}
    cgroup: host
    tmpfs:
      - /run
      - /run/lock
      - /tmp
    env_file:
      - .env
    environment:
      - TZ=Asia/Shanghai
      - PYTHONWARNINGS=ignore::DeprecationWarning
      - SALT_MASTER_ID=salt-master-1
      - SALT_MASTER_ROLE=primary
      - AI_INFRA_BACKEND_URL=${BACKEND_URL:-http://backend:8082}
      - DEBUG_MODE=${DEBUG_MODE:-false}
      - SALT_API_USERNAME=${SALT_API_USERNAME:-saltapi}
      - SALT_API_PASSWORD=${SALT_API_PASSWORD:-aiinfra-salt}
      - START_LOCAL_MINION=${SALT_START_LOCAL_MINION:-false}
    ports:
      - "4505:4505"
      - "4506:4506"
    expose:
      - "8002"
    volumes:
      - {{CGROUP_MOUNT}}
      - salt_master_1_cache:/var/cache/salt
      - salt_logs:/var/log/salt
      - salt_keys:/etc/salt/pki
      - salt_states:/srv/salt
      - salt_pillar:/srv/pillar
    networks:
      ai-infra-network:
        aliases:
          - saltstack
          - salt-master
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "sh", "-c", "pgrep -f salt-master && pgrep -f salt-api && nc -z 127.0.0.1 8002"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 90s
    restart: unless-stopped

  # SaltStack Master 2 (备用节点 - 故障转移)
  salt-master-2:
    image: ai-infra-saltstack:{{IMAGE_TAG}}
    build:
      context: .
      dockerfile: ./src/saltstack/Dockerfile
      args:
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-salt-master-2
    hostname: salt-master-2
    privileged: true
    security_opt:
      - seccomp:unconfined
    # cgroup 配置: 支持 cgroupv1 和 cgroupv2
    # 自动检测: CGROUP_VERSION={{CGROUP_VERSION}}
    cgroup: host
    tmpfs:
      - /run
      - /run/lock
      - /tmp
    env_file:
      - .env
    environment:
      - TZ=Asia/Shanghai
      - PYTHONWARNINGS=ignore::DeprecationWarning
      - SALT_MASTER_ID=salt-master-2
      - SALT_MASTER_ROLE=secondary
      - AI_INFRA_BACKEND_URL=${BACKEND_URL:-http://backend:8082}
      - DEBUG_MODE=${DEBUG_MODE:-false}
      - SALT_API_USERNAME=${SALT_API_USERNAME:-saltapi}
      - SALT_API_PASSWORD=${SALT_API_PASSWORD:-aiinfra-salt}
      - START_LOCAL_MINION=false
    ports:
      - "4507:4505"
      - "4508:4506"
    expose:
      - "8002"
    volumes:
      - {{CGROUP_MOUNT}}
      - salt_master_2_cache:/var/cache/salt
      - salt_logs:/var/log/salt
      - salt_keys:/etc/salt/pki
      - salt_states:/srv/salt
      - salt_pillar:/srv/pillar
    networks:
      ai-infra-network:
        aliases:
          - salt-master-backup
    depends_on:
      salt-master-1:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "sh", "-c", "pgrep -f salt-master && pgrep -f salt-api && nc -z 127.0.0.1 8002"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 90s
    restart: unless-stopped
    profiles:
      - ha

  # SLURM Master 服务
  slurm-master:
    image: ai-infra-slurm-master:{{IMAGE_TAG}}
    build:
      context: ./src/slurm-master
      dockerfile: Dockerfile
      args:
        VERSION: {{IMAGE_TAG}}
        APPHUB_URL: http://${EXTERNAL_HOST}:${APPHUB_PORT:-8090}
    container_name: ai-infra-slurm-master
    hostname: slurm-master
    privileged: true
    security_opt:
      - seccomp:unconfined
    # cgroup 配置：让容器拥有自己的 cgroup 命名空间，这对 systemd 运行至关重要
    # 自动检测: CGROUP_VERSION={{CGROUP_VERSION}}
    cgroup: host
    tmpfs:
      - /run
      - /run/lock
      - /tmp
    dns:
      - 223.5.5.5
      - 8.8.8.8
    env_file:
      - .env
    environment:
      - TZ=Asia/Shanghai
      # SLURM 集群配置
      - SLURM_CLUSTER_NAME=${SLURM_CLUSTER_NAME:-ai-infra-cluster}
      - SLURM_CONTROL_MACHINE=${SLURM_CONTROLLER_HOST:-slurm-master}
      - SLURM_CONTROLLER_PORT=${SLURM_CONTROLLER_PORT:-6817}
      - SLURM_SLURMDBD_HOST=${SLURM_SLURMDBD_HOST:-slurm-master}
      - SLURM_SLURMDBD_PORT=${SLURM_SLURMDBD_PORT:-6818}
      # SLURM 数据库配置
      - SLURM_DB_HOST=${SLURM_DB_HOST:-mysql}
      - SLURM_DB_PORT=${SLURM_DB_PORT:-3306}
      - SLURM_DB_NAME=${SLURM_DB_NAME:-slurm_acct_db}
      - SLURM_DB_USER=${SLURM_DB_USER:-slurm}
      - SLURM_DB_PASSWORD=${SLURM_DB_PASSWORD:-slurm123}
      # SLURM 认证配置
      - SLURM_AUTH_TYPE=${SLURM_AUTH_TYPE:-auth/munge}
      - SLURM_MUNGE_KEY=${SLURM_MUNGE_KEY:-ai-infra-slurm-munge-key-dev}
      # SLURM 集群节点配置
      - SLURM_PARTITION_NAME=${SLURM_PARTITION_NAME:-compute}
      - SLURM_DEFAULT_PARTITION=${SLURM_DEFAULT_PARTITION:-compute}
      - SLURM_NODE_PREFIX=${SLURM_NODE_PREFIX:-compute}
      - SLURM_NODE_COUNT=${SLURM_NODE_COUNT:-3}
      # SLURM 测试节点配置
      - SLURM_TEST_NODES=${SLURM_TEST_NODES:-}
      - SLURM_TEST_NODE_CPUS=${SLURM_TEST_NODE_CPUS:-4}
      - SLURM_TEST_NODE_MEMORY=${SLURM_TEST_NODE_MEMORY:-8192}
      # SLURM 作业配置
      - SLURM_MAX_JOB_COUNT=${SLURM_MAX_JOB_COUNT:-10000}
      - SLURM_MAX_ARRAY_SIZE=${SLURM_MAX_ARRAY_SIZE:-1000}
      - SLURM_DEFAULT_TIME_LIMIT=${SLURM_DEFAULT_TIME_LIMIT:-01:00:00}
      - SLURM_MAX_TIME_LIMIT=${SLURM_MAX_TIME_LIMIT:-24:00:00}
      # 后端服务连接
      - AI_INFRA_BACKEND_URL=${BACKEND_URL:-http://backend:8082}
      - DEBUG_MODE=${DEBUG_MODE:-false}
    expose:
      - "6817"  # SLURM Controller
      - "6818"  # SLURM Database Daemon
    ports:
      - "6817:6817"  # SLURM Controller 外部访问
      - "6818:6818"  # SLURM Database Daemon 外部访问
    volumes:
      - {{CGROUP_MOUNT}}
      - slurm_master_data:/var/lib/slurm
      - slurm_master_logs:/var/log/slurm
      - slurm_master_spool:/var/spool/slurm
      - slurm_munge_data:/var/lib/munge
      - ./shared:/srv/shared:rw
    depends_on:
      mysql:
        condition: service_healthy
      backend:
        condition: service_healthy
      backend-init:
        condition: service_completed_successfully
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "/usr/local/bin/healthcheck.sh"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 90s
    restart: unless-stopped

  # Nginx 反向代理服务 - 自定义镜像版本
  nginx:
    image: ai-infra-nginx:{{IMAGE_TAG}}
    build:
      context: .
      dockerfile: src/nginx/Dockerfile
      args:
        DEBUG_MODE: ${DEBUG_MODE:-false}
        BUILD_ENV: ${BUILD_ENV:-production}
        VERSION: {{IMAGE_TAG}}
    container_name: ai-infra-nginx
    ports:
      - "${BIND_HOST:-0.0.0.0}:${EXTERNAL_PORT}:80"
      - "${BIND_HOST:-0.0.0.0}:${HTTPS_PORT:-8443}:443"
      - "${BIND_HOST:-0.0.0.0}:${DEBUG_PORT}:8001"
    volumes:
      - nginx_logs:/var/log/nginx
      # 挂载配置文件以支持热更新 (模板渲染后的文件)
      - ./src/nginx/conf.d/includes:/etc/nginx/conf.d/includes:rw
    env_file:
      - .env
    environment:
      - DEBUG_MODE=${DEBUG_MODE:-false}
      - BUILD_ENV=${BUILD_ENV:-production}
      - BACKEND_HOST=${BACKEND_HOST:-backend}
      - BACKEND_PORT=${BACKEND_PORT:-8082}
      - FRONTEND_HOST=${FRONTEND_HOST:-frontend}
      - FRONTEND_PORT=${FRONTEND_PORT:-80}
      - JUPYTERHUB_HOST=${JUPYTERHUB_HOST:-jupyterhub}
      - JUPYTERHUB_PORT=${JUPYTERHUB_PORT:-8000}
      - EXTERNAL_HOST=${EXTERNAL_HOST:-localhost}
      - EXTERNAL_PORT=${EXTERNAL_PORT:-8080}
      - HTTPS_PORT=${HTTPS_PORT:-8443}
      - EXTERNAL_SCHEME=${EXTERNAL_SCHEME:-http}
      - ENABLE_TLS=${ENABLE_TLS:-false}
      - TZ=Asia/Shanghai
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      openldap:
        condition: service_healthy
      seaweedfs-filer:
        condition: service_healthy
      gitea:
        condition: service_healthy
      frontend:
        condition: service_healthy
      backend:
        condition: service_healthy
      jupyterhub:
        condition: service_healthy
      salt-master-1:
        condition: service_healthy
      slurm-master:
        condition: service_healthy
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 15s
      retries: 5
      start_period: 90s
    restart: unless-stopped

  # Redis 监控界面 (可选)
  # 注意：RedisInsight 2.68+ 已原生支持多架构，无需指定 platform
  redis-insight:
    image: redis/redisinsight:{{REDISINSIGHT_VERSION}}
    container_name: ai-infra-redis-insight
    environment:
      TZ: Asia/Shanghai
    expose:
      - "8001"
    depends_on:
      - redis
    networks:
      - ai-infra-network
    restart: unless-stopped


  # Gitea 代码托管服务（可选，供门户内嵌）
  gitea:
    image: ai-infra-gitea:{{IMAGE_TAG}}
    build:
      context: ./src/gitea
      dockerfile: Dockerfile
    container_name: ai-infra-gitea
    env_file:
      - .env
    environment:
      USER_UID: "${USER_UID:-1000}"
      USER_GID: "${USER_GID:-1000}"
      ROOT_URL: "${ROOT_URL}"
      SUBURL: "${SUBURL:-/gitea}"
      STATIC_URL_PREFIX: "${STATIC_URL_PREFIX:-/gitea}"
      GITEA__server__ROOT_URL: "${ROOT_URL}"
      GITEA__server__SUBURL: "${SUBURL:-/gitea}"
      # 使用 GITEA__ 前缀直接设置 server 配置，避免嵌套变量替换问题
      GITEA__server__DOMAIN: "${EXTERNAL_HOST}"
      GITEA__server__SSH_DOMAIN: "${EXTERNAL_HOST}"
      PROTOCOL: "${GITEA_PROTOCOL:-http}"
      HTTP_PORT: "${GITEA_HTTP_PORT:-3000}"
      GITEA_DB_TYPE: "${GITEA_DB_TYPE:-postgres}"
      GITEA_DB_HOST: "${GITEA_DB_HOST:-postgres:5432}"
      GITEA_DB_NAME: "${GITEA_DB_NAME:-gitea}"
      # 注意：以下两行必须使用6个空格缩进，保持与其他 environment 项对齐，否则会导致 YAML 解析错误。
      GITEA_DB_USER: "${GITEA_DB_USER:-gitea}"
      GITEA_DB_PASSWD: "${GITEA_DB_PASSWD:-gitea-password}"
      POSTGRES_USER: "${POSTGRES_USER:-postgres}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD:-postgres}"
      DISABLE_REGISTRATION: "${DISABLE_REGISTRATION:-false}"
      REVERSE_PROXY_TRUSTED_PROXIES: "${REVERSE_PROXY_TRUSTED_PROXIES:-0.0.0.0/0,::/0}"
      GITEA__auth__REVERSE_PROXY_AUTHENTICATION_USER: "X-WEBAUTH-USER"
      GITEA__auth__REVERSE_PROXY_AUTHENTICATION_EMAIL: "X-WEBAUTH-EMAIL"
      GITEA__auth__REVERSE_PROXY_AUTHENTICATION_FULL_NAME: "X-WEBAUTH-FULLNAME"
      GITEA__auth__PROXY_ENABLED: "true"
      GITEA__auth__PROXY_HEADER_NAME: "X-WEBAUTH-USER"
      GITEA__auth__PROXY_EMAIL_HEADER: "X-WEBAUTH-EMAIL"
      GITEA__auth__PROXY_FULL_NAME_HEADER: "X-WEBAUTH-FULLNAME"
      GITEA__security__REVERSE_PROXY_LIMIT: "1"
      GITEA__security__REVERSE_PROXY_TRUSTED_PROXIES: "0.0.0.0/0,::/0"
      # Set initial admin identity to 'admin' user
      INITIAL_ADMIN_USERNAME: "${GITEA_ALIAS_ADMIN_TO:-admin}"
      # Admin bootstrap via reverse-proxy SSO only; admin user will be default admin
      GITEA__storage__STORAGE_TYPE: "${GITEA_STORAGE:-local}"
      DATA_PATH: "${GITEA_DATA_PATH:-/data/gitea}"
      # SeaweedFS S3 兼容存储配置 (用于 Gitea 附件等)
      MINIO_ENDPOINT: "${SEAWEEDFS_FILER_HOST:-seaweedfs-filer}:${SEAWEEDFS_S3_PORT:-8333}"
      MINIO_BUCKET: "${SEAWEEDFS_BUCKET_GITEA:-gitea}"
      MINIO_USE_SSL: "${SEAWEEDFS_USE_SSL:-true}"
      MINIO_LOCATION: "${SEAWEEDFS_REGION:-us-east-1}"
      MINIO_ACCESS_KEY: "${SEAWEEDFS_ACCESS_KEY:-admin}"
      MINIO_SECRET_KEY: "${SEAWEEDFS_SECRET_KEY:-admin123456}"
      TZ: "${TZ:-Asia/Shanghai}"
    expose:
      - "3000"
      - "22"
    ports:
      # Debug: expose internal 3000 on host for direct backend access (bypass Nginx)
      - "${BIND_HOST:-0.0.0.0}:${GITEA_EXTERNAL_PORT}:3000"
    volumes:
      - gitea_data:/data
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      openldap:
        condition: service_healthy
    networks:
      - ai-infra-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/"]
      interval: 30s
      timeout: 15s
      retries: 5
      start_period: 60s
    restart: unless-stopped

  # SeaweedFS Master - 管理集群元数据
  seaweedfs-master:
    image: {{SEAWEEDFS_IMAGE}}:{{SEAWEEDFS_VERSION}}
    container_name: ai-infra-seaweedfs-master
    user: root
    command: master -ip=seaweedfs-master -ip.bind=0.0.0.0 -port=9333 -mdir=/data -volumeSizeLimitMB=1024
    env_file:
      - .env
    environment:
      - TZ=Asia/Shanghai
    expose:
      - "9333"
      - "19333"
    volumes:
      - seaweedfs_master_data:/data
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:9333/cluster/status"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # SeaweedFS Volume - 存储实际数据
  seaweedfs-volume:
    image: {{SEAWEEDFS_IMAGE}}:{{SEAWEEDFS_VERSION}}
    container_name: ai-infra-seaweedfs-volume
    user: root
    command: volume -ip=seaweedfs-volume -ip.bind=0.0.0.0 -port=8080 -dir=/data -max=100 -mserver=seaweedfs-master:9333 -publicUrl=seaweedfs-volume:8080
    env_file:
      - .env
    environment:
      - TZ=Asia/Shanghai
    expose:
      - "8080"
      - "18080"
    volumes:
      - seaweedfs_volume_data:/data
    depends_on:
      seaweedfs-master:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:8080/status"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # SeaweedFS Filer - 提供文件系统接口和 S3 API
  seaweedfs-filer:
    image: {{SEAWEEDFS_IMAGE}}:{{SEAWEEDFS_VERSION}}
    container_name: ai-infra-seaweedfs-filer
    user: root
    command: filer -ip=seaweedfs-filer -ip.bind=0.0.0.0 -port=8888 -master=seaweedfs-master:9333 -s3 -s3.port=8333 -s3.config=/etc/seaweedfs/s3.json
    env_file:
      - .env
    environment:
      - TZ=Asia/Shanghai
    expose:
      - "8888"  # Filer HTTP API
      - "18888" # Filer gRPC
      - "8333"  # S3 API
    volumes:
      - seaweedfs_filer_data:/data
      - ./config/seaweedfs/s3.json:/etc/seaweedfs/s3.json:ro
    depends_on:
      seaweedfs-master:
        condition: service_healthy
      seaweedfs-volume:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "-", "http://localhost:8888/?pretty=y"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # AppHub - 二进制包仓库服务 (DEB/RPM packages)
  apphub:
    build:
      context: .
      dockerfile: src/apphub/Dockerfile
    image: ${PRIVATE_REGISTRY}ai-infra-apphub:{{IMAGE_TAG}}
    container_name: ai-infra-apphub
    env_file:
      - .env
    ports:
      - "${BIND_HOST:-0.0.0.0}:${APPHUB_PORT}:80"
    volumes:
      - apphub_logs:/var/log/nginx
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # VictoriaMetrics - 时序数据库 (Prometheus 兼容)
  victoriametrics:
    image: victoriametrics/victoria-metrics:{{VICTORIAMETRICS_VERSION}}
    container_name: ai-infra-victoriametrics
    hostname: victoriametrics
    environment:
      TZ: {{TZ}}
    ports:
      - "${BIND_HOST:-0.0.0.0}:${VICTORIAMETRICS_PORT:-8428}:8428"
    volumes:
      - victoriametrics_data:/victoria-metrics-data
    command:
      - "--storageDataPath=/victoria-metrics-data"
      - "--httpListenAddr=:8428"
      - "--retentionPeriod=${VICTORIAMETRICS_RETENTION:-30d}"
      - "--search.latencyOffset=0s"
      - "--search.maxUniqueTimeseries=300000"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://127.0.0.1:8428/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # Nightingale - 监控告警系统
  nightingale:
    build:
      context: .
      dockerfile: src/nightingale/Dockerfile
    image: ${PRIVATE_REGISTRY}ai-infra-nightingale:{{IMAGE_TAG}}
    container_name: ai-infra-nightingale
    hostname: nightingale
    env_file:
      - .env
    environment:
      GIN_MODE: release
      TZ: Asia/Shanghai
      # PostgreSQL 配置
      POSTGRES_HOST: ${POSTGRES_HOST:-postgres}
      POSTGRES_PORT: ${POSTGRES_PORT:-5432}
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
      N9E_DB_NAME: ${N9E_DB_NAME:-nightingale}
      # Redis 配置
      REDIS_HOST: ${REDIS_HOST:-redis}
      REDIS_PORT: ${REDIS_PORT:-6379}
      REDIS_PASSWORD: ${REDIS_PASSWORD}
    ports:
      - "${BIND_HOST:-0.0.0.0}:${NIGHTINGALE_PORT:-17000}:17000"  # HTTP API
      - "${BIND_HOST:-0.0.0.0}:${NIGHTINGALE_ALERT_PORT:-19000}:19000"  # Alert engine
    volumes:
      - ./src/nightingale/etc:/app/etc:ro
      - nightingale_data:/app/data
      - nightingale_logs:/app/logs
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      victoriametrics:
        condition: service_healthy
      backend-init:
        condition: service_completed_successfully
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:17000/nightingale/"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # ==================== Keycloak IAM 服务 ====================
  # Keycloak 统一身份认证服务
  # 提供 SSO、OIDC、SAML 等认证协议支持
  # 启动方式: docker-compose --profile keycloak up -d
  keycloak:
    image: ai-infra-keycloak:{{IMAGE_TAG}}
    build:
      context: ./src/keycloak
      dockerfile: Dockerfile.tpl
      args:
        BASE_IMAGE_REGISTRY: ${BASE_IMAGE_REGISTRY:-}
        KEYCLOAK_VERSION: ${KEYCLOAK_VERSION:-26.0}
    container_name: ai-infra-keycloak
    profiles:
      - keycloak
      - sso
      - full
    env_file:
      - .env
    environment:
      TZ: ${TZ:-Asia/Shanghai}
      # 数据库配置
      KC_DB: postgres
      KC_DB_URL: jdbc:postgresql://${POSTGRES_HOST:-postgres}:${POSTGRES_PORT:-5432}/${KEYCLOAK_DB_NAME:-keycloak}
      KC_DB_USERNAME: ${KEYCLOAK_DB_USER:-keycloak}
      KC_DB_PASSWORD: ${KEYCLOAK_DB_PASSWORD:-keycloak}
      # 主机名配置
      KC_HOSTNAME: ${EXTERNAL_HOST:-localhost}
      KC_HOSTNAME_PORT: ${EXTERNAL_PORT:-8080}
      KC_HOSTNAME_STRICT: "false"
      KC_HOSTNAME_STRICT_HTTPS: "false"
      KC_HTTP_ENABLED: "true"
      KC_HTTP_RELATIVE_PATH: /auth
      # 代理配置
      KC_PROXY_HEADERS: xforwarded
      # 健康检查
      KC_HEALTH_ENABLED: "true"
      KC_METRICS_ENABLED: "true"
      # 管理员配置
      KEYCLOAK_ADMIN: ${KEYCLOAK_ADMIN:-admin}
      KEYCLOAK_ADMIN_PASSWORD: ${KEYCLOAK_ADMIN_PASSWORD:-admin}
      # 客户端密钥 (供 realm 导入使用)
      KEYCLOAK_BACKEND_CLIENT_SECRET: ${KEYCLOAK_BACKEND_CLIENT_SECRET:-backend-client-secret}
      KEYCLOAK_GITEA_CLIENT_SECRET: ${KEYCLOAK_GITEA_CLIENT_SECRET:-gitea-client-secret}
      KEYCLOAK_N9E_CLIENT_SECRET: ${KEYCLOAK_N9E_CLIENT_SECRET:-n9e-client-secret}
      KEYCLOAK_ARGOCD_CLIENT_SECRET: ${KEYCLOAK_ARGOCD_CLIENT_SECRET:-argocd-client-secret}
      KEYCLOAK_JUPYTERHUB_CLIENT_SECRET: ${KEYCLOAK_JUPYTERHUB_CLIENT_SECRET:-jupyterhub-client-secret}
      # 外部访问地址 (用于 realm 配置)
      EXTERNAL_SCHEME: ${EXTERNAL_SCHEME:-https}
      EXTERNAL_HOST: ${EXTERNAL_HOST:-localhost}
      EXTERNAL_PORT: ${EXTERNAL_PORT:-8080}
      HTTPS_PORT: ${HTTPS_PORT:-8443}
      # LDAP 配置 (用于 LDAP Federation)
      LDAP_SERVER: ${LDAP_SERVER:-openldap}
      LDAP_PORT: ${LDAP_PORT:-389}
      LDAP_BASE_DN: ${LDAP_BASE_DN:-dc=ai-infra,dc=com}
      LDAP_BIND_DN: ${LDAP_BIND_DN:-cn=admin,dc=ai-infra,dc=com}
      LDAP_ADMIN_PASSWORD: ${LDAP_ADMIN_PASSWORD}
      LDAP_USER_BASE: ${LDAP_USER_BASE:-ou=users,dc=ai-infra,dc=com}
    expose:
      - "8080"
      - "8443"
    ports:
      - "${BIND_HOST:-0.0.0.0}:${KEYCLOAK_HTTP_PORT:-8180}:8080"
      - "${BIND_HOST:-0.0.0.0}:${KEYCLOAK_HTTPS_PORT:-8543}:8443"
    volumes:
      - keycloak_data:/opt/keycloak/data
    depends_on:
      postgres:
        condition: service_healthy
      openldap:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "exec 3<>/dev/tcp/127.0.0.1/8080 && echo -e 'GET /auth/health/ready HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n' >&3 && cat <&3 | grep -q '200 OK'"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 120s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # ==================== ArgoCD GitOps 服务 ====================
  # ArgoCD 持续部署服务
  # 提供 GitOps 风格的 Kubernetes 应用部署
  # 启动方式: docker-compose --profile argocd up -d
  argocd-server:
    image: ai-infra-argocd:{{IMAGE_TAG}}
    build:
      context: ./src/argocd
      dockerfile: Dockerfile.tpl
      args:
        BASE_IMAGE_REGISTRY: ${BASE_IMAGE_REGISTRY:-}
        ARGOCD_VERSION: ${ARGOCD_VERSION:-v2.13.3}
    container_name: ai-infra-argocd-server
    profiles:
      - argocd
      - gitops
      - full
    env_file:
      - .env
    environment:
      TZ: ${TZ:-Asia/Shanghai}
      # ArgoCD 服务器配置
      ARGOCD_SERVER_INSECURE: "true"
      ARGOCD_SERVER_BASEHREF: /argocd
      ARGOCD_SERVER_ROOTPATH: /argocd
      ARGOCD_SERVER_DISABLE_AUTH: "false"
      # Redis 配置 (ArgoCD 使用 Redis 作为缓存)
      REDIS_SERVER: ${REDIS_HOST:-redis}:${REDIS_PORT:-6379}
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      # Dex/OIDC 配置 (Keycloak 集成)
      ARGOCD_DEX_SERVER_DISABLE_TLS: "true"
      # 仓库配置 (Gitea 集成)
      ARGOCD_REPO_SERVER_STRICT_TLS: "false"
      # Keycloak OIDC 配置
      KEYCLOAK_ISSUER: ${EXTERNAL_SCHEME:-https}://${EXTERNAL_HOST:-localhost}:${EXTERNAL_PORT:-8080}/auth/realms/ai-infra
      KEYCLOAK_ARGOCD_CLIENT_ID: argocd
      KEYCLOAK_ARGOCD_CLIENT_SECRET: ${KEYCLOAK_ARGOCD_CLIENT_SECRET:-argocd-client-secret}
      # 外部访问地址
      EXTERNAL_SCHEME: ${EXTERNAL_SCHEME:-https}
      EXTERNAL_HOST: ${EXTERNAL_HOST:-localhost}
      EXTERNAL_PORT: ${EXTERNAL_PORT:-8080}
      HTTPS_PORT: ${HTTPS_PORT:-8443}
    command: ["argocd-server", "--insecure", "--basehref", "/argocd", "--rootpath", "/argocd"]
    expose:
      - "8080"
      - "8083"
    ports:
      - "${BIND_HOST:-0.0.0.0}:${ARGOCD_HTTP_PORT:-8280}:8080"
    volumes:
      - argocd_data:/home/argocd
      - ./src/argocd/argocd-cm.yaml:/home/argocd/argocd-cm.yaml:ro
      - ./src/argocd/argocd-rbac-cm.yaml:/home/argocd/argocd-rbac-cm.yaml:ro
    depends_on:
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/argocd/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # ArgoCD Repo Server
  argocd-repo-server:
    image: quay.io/argoproj/argocd:${ARGOCD_VERSION:-v2.13.3}
    container_name: ai-infra-argocd-repo-server
    profiles:
      - argocd
      - gitops
      - full
    environment:
      TZ: ${TZ:-Asia/Shanghai}
      REDIS_SERVER: ${REDIS_HOST:-redis}:${REDIS_PORT:-6379}
      REDIS_PASSWORD: ${REDIS_PASSWORD}
    command: ["argocd-repo-server"]
    expose:
      - "8081"
    volumes:
      - argocd_repo_data:/tmp
    depends_on:
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # ArgoCD Application Controller
  argocd-application-controller:
    image: quay.io/argoproj/argocd:${ARGOCD_VERSION:-v2.13.3}
    container_name: ai-infra-argocd-controller
    profiles:
      - argocd
      - gitops
      - full
    environment:
      TZ: ${TZ:-Asia/Shanghai}
      REDIS_SERVER: ${REDIS_HOST:-redis}:${REDIS_PORT:-6379}
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      ARGOCD_RECONCILIATION_TIMEOUT: 180s
    command: ["argocd-application-controller", "--repo-server", "argocd-repo-server:8081"]
    volumes:
      - argocd_controller_data:/tmp
    depends_on:
      - argocd-repo-server
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - ai-infra-network
    restart: unless-stopped

  # SafeLine WAF Services (Optional - use --profile safeline to start)
  # 启动方式: docker-compose --profile safeline up -d
  # 或先运行: ./build.sh init-safeline
  safeline-postgres:
    container_name: safeline-pg
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-postgres{{SAFELINE_ARCH_SUFFIX}}:15.2
    profiles:
      - safeline
    volumes:
      - ${SAFELINE_DIR:-./data/safeline}/resources/postgres/data:/var/lib/postgresql/data
      - /etc/localtime:/etc/localtime:ro
    environment:
      - POSTGRES_USER=safeline-ce
      - POSTGRES_PASSWORD=${SAFELINE_POSTGRES_PASSWORD}
    networks:
      safeline-ce:
        ipv4_address: ${SAFELINE_SUBNET_PREFIX}.2
    command: ["postgres", "-c", "max_connections=500"]

  safeline-mgt:
    container_name: safeline-mgt
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-mgt{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
    profiles:
      - safeline
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - ${SAFELINE_DIR:-./data/safeline}/resources/mgt:/app/data
      - ${SAFELINE_DIR:-./data/safeline}/logs/nginx:/app/log/nginx:z
      - ${SAFELINE_DIR:-./data/safeline}/resources/sock:/app/sock
      - ${SAFELINE_DIR:-./data/safeline}/run:/app/run
    ports:
      - "${SAFELINE_MGT_PORT}:1443"
    healthcheck:
      test: curl -k -f https://localhost:1443/api/open/health
    environment:
      - MGT_PG=postgres://safeline-ce:${SAFELINE_POSTGRES_PASSWORD}@safeline-pg/safeline-ce?sslmode=disable
    depends_on:
      - safeline-postgres
      - safeline-fvm
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
    networks:
      safeline-ce:
        ipv4_address: ${SAFELINE_SUBNET_PREFIX}.4

  safeline-detector:
    container_name: safeline-detector
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-detector{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
    profiles:
      - safeline
    volumes:
      - ${SAFELINE_DIR:-./data/safeline}/resources/detector:/resources/detector
      - ${SAFELINE_DIR:-./data/safeline}/logs/detector:/logs/detector
      - /etc/localtime:/etc/localtime:ro
    environment:
      - LOG_DIR=/logs/detector
    networks:
      safeline-ce:
        ipv4_address: ${SAFELINE_SUBNET_PREFIX}.5

  safeline-tengine:
    container_name: safeline-tengine
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-tengine{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
    profiles:
      - safeline
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - /etc/resolv.conf:/etc/resolv.conf:ro
      - ${SAFELINE_DIR:-./data/safeline}/resources/nginx:/etc/nginx
      - ${SAFELINE_DIR:-./data/safeline}/resources/detector:/resources/detector
      - ${SAFELINE_DIR:-./data/safeline}/resources/chaos:/resources/chaos
      - ${SAFELINE_DIR:-./data/safeline}/logs/nginx:/var/log/nginx:z
      - ${SAFELINE_DIR:-./data/safeline}/resources/cache:/usr/local/nginx/cache
      - ${SAFELINE_DIR:-./data/safeline}/resources/sock:/app/sock
    environment:
      - TCD_MGT_API=https://${SAFELINE_SUBNET_PREFIX}.4:1443/api/open/publish/server
      - TCD_SNSERVER=${SAFELINE_SUBNET_PREFIX}.5:8000
      - CHAOS_ADDR=${SAFELINE_SUBNET_PREFIX}.10
    ulimits:
      nofile: 131072
    depends_on:
      - safeline-mgt
      - safeline-detector
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
    # 使用 host 网络模式，直接监听宿主机 80/443 端口
    network_mode: host

  safeline-luigi:
    container_name: safeline-luigi
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-luigi{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
    profiles:
      - safeline
    environment:
      - MGT_IP=${SAFELINE_SUBNET_PREFIX}.4
      - LUIGI_PG=postgres://safeline-ce:${SAFELINE_POSTGRES_PASSWORD}@safeline-pg/safeline-ce?sslmode=disable
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - ${SAFELINE_DIR:-./data/safeline}/resources/luigi:/app/data
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
    depends_on:
      - safeline-detector
      - safeline-mgt
    networks:
      safeline-ce:
        ipv4_address: ${SAFELINE_SUBNET_PREFIX}.7

  safeline-fvm:
    container_name: safeline-fvm
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-fvm{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
    profiles:
      - safeline
    volumes:
      - /etc/localtime:/etc/localtime:ro
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "5"
    networks:
      safeline-ce:
        ipv4_address: ${SAFELINE_SUBNET_PREFIX}.8

  safeline-chaos:
    container_name: safeline-chaos
    restart: always
    image: {{SAFELINE_IMAGE_PREFIX}}/safeline-chaos{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
    profiles:
      - safeline
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "10"
    environment:
      - DB_ADDR=postgres://safeline-ce:${SAFELINE_POSTGRES_PASSWORD}@safeline-pg/safeline-ce?sslmode=disable
    volumes:
      - ${SAFELINE_DIR:-./data/safeline}/resources/sock:/app/sock
      - ${SAFELINE_DIR:-./data/safeline}/resources/chaos:/app/chaos
    networks:
      safeline-ce:
        ipv4_address: ${SAFELINE_SUBNET_PREFIX}.10

volumes:
  postgres_data:
    name: ai-infra-postgres-data
  mysql_data:
    name: ai-infra-mysql-data
  redis_data:
    name: ai-infra-redis-data
  kafka_data:
    name: ai-infra-kafka-data
  kafka_logs:
    name: ai-infra-kafka-logs
  ldap_data:
    name: ai-infra-ldap-data
  ldap_config:
    name: ai-infra-ldap-config
  jupyterhub_data:
    name: ai-infra-jupyterhub-data
  jupyterhub_notebooks:
    name: ai-infra-jupyterhub-notebooks
  nginx_logs:
    name: ai-infra-nginx-logs
  salt_data:
    name: ai-infra-salt-data
  salt_master_1_cache:
    name: ai-infra-salt-master-1-cache
  salt_master_2_cache:
    name: ai-infra-salt-master-2-cache
  salt_logs:
    name: ai-infra-salt-logs
  salt_keys:
    name: ai-infra-salt-keys
  salt_states:
    name: ai-infra-salt-states
  salt_pillar:
    name: ai-infra-salt-pillar
  slurm_master_data:
    name: ai-infra-slurm-master-data
  slurm_master_logs:
    name: ai-infra-slurm-master-logs
  slurm_master_spool:
    name: ai-infra-slurm-master-spool
  slurm_munge_data:
    name: ai-infra-slurm-munge-data
  gitea_data:
    name: ai-infra-gitea-data
  seaweedfs_master_data:
    name: ai-infra-seaweedfs-master-data
  seaweedfs_volume_data:
    name: ai-infra-seaweedfs-volume-data
  seaweedfs_filer_data:
    name: ai-infra-seaweedfs-filer-data
  apphub_logs:
    name: ai-infra-apphub-logs
  nightingale_data:
    name: ai-infra-nightingale-data
  nightingale_logs:
    name: ai-infra-nightingale-logs
  victoriametrics_data:
    name: ai-infra-victoriametrics-data
  keycloak_data:
    name: ai-infra-keycloak-data
  argocd_data:
    name: ai-infra-argocd-data
  argocd_repo_data:
    name: ai-infra-argocd-repo-data
  argocd_controller_data:
    name: ai-infra-argocd-controller-data

networks:
  ai-infra-network:
    name: ai-infra-network
    driver: bridge
    ipam:
      driver: default
      config:
        - subnet: 172.16.238.0/24
          gateway: 172.16.238.1
  safeline-ce:
    name: safeline-ce
    driver: bridge
    ipam:
      driver: default
      config:
        - gateway: ${SAFELINE_SUBNET_PREFIX}.1
          subnet: ${SAFELINE_SUBNET_PREFIX}.0/24
    driver_opts:
      com.docker.network.bridge.name: safeline-ce
