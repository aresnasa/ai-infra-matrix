# Keycloak IAM 服务 - 统一身份认证管理
# 版本: 26.0
# 用途: 为 AI Infrastructure Matrix 提供 SSO 单点登录、OIDC/SAML 认证服务
# 支持架构: linux/amd64, linux/arm64

ARG KEYCLOAK_VERSION=26.0

FROM quay.io/keycloak/keycloak:${KEYCLOAK_VERSION} AS builder

# 设置工作目录
WORKDIR /opt/keycloak

# 构建优化的 Keycloak 镜像
# 启用健康检查和指标端点
ENV KC_HEALTH_ENABLED=true
ENV KC_METRICS_ENABLED=true

# 数据库配置（使用 PostgreSQL）
ENV KC_DB=postgres

# 构建优化版本
RUN /opt/keycloak/bin/kc.sh build \
    --db=postgres \
    --features=docker,admin-fine-grained-authz,token-exchange,organization

# ==================== 生产镜像 ====================
FROM quay.io/keycloak/keycloak:${KEYCLOAK_VERSION}

# 从构建阶段复制优化后的文件
COPY --from=builder /opt/keycloak/ /opt/keycloak/

# 复制自定义主题和配置
# 注意：构建上下文为项目根目录，所以使用 src/keycloak/ 前缀
COPY src/keycloak/themes/ /opt/keycloak/themes/
COPY src/keycloak/realm-export/ /opt/keycloak/data/import/

# 设置工作目录
WORKDIR /opt/keycloak

# 环境变量配置
# 数据库连接配置（非敏感配置，密码在运行时通过环境变量传入）
ENV KC_DB=postgres
ENV KC_DB_URL_HOST=postgres
ENV KC_DB_URL_PORT=5432
ENV KC_DB_URL_DATABASE=keycloak
ENV KC_DB_USERNAME=keycloak
# KC_DB_PASSWORD 通过运行时环境变量传入

# 主机名配置
ENV KC_HOSTNAME_STRICT=false
ENV KC_HOSTNAME_STRICT_HTTPS=false
ENV KC_HTTP_ENABLED=true
ENV KC_HTTP_PORT=8080
ENV KC_HTTPS_PORT=8443
ENV KC_HTTP_RELATIVE_PATH=/auth

# 代理配置（反向代理模式）
ENV KC_PROXY_HEADERS=xforwarded

# 健康检查和指标
ENV KC_HEALTH_ENABLED=true
ENV KC_METRICS_ENABLED=true

# 管理员用户名（密码通过运行时环境变量 KEYCLOAK_ADMIN_PASSWORD 传入）
ENV KEYCLOAK_ADMIN=admin
# KEYCLOAK_ADMIN_PASSWORD 通过运行时环境变量传入

# 时区设置
ENV TZ=Asia/Shanghai

# 端口暴露
EXPOSE 8080 8443

# 启动命令 - 使用 start 模式
ENTRYPOINT ["/opt/keycloak/bin/kc.sh"]
CMD ["start", "--optimized", "--import-realm"]
