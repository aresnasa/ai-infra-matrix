# =============================================================================
# Prometheus Server Dockerfile Template
# 用于 AI-Infra-Matrix 监控数据采集
# =============================================================================

ARG PROMETHEUS_VERSION={{PROMETHEUS_VERSION}}

# 使用官方 Prometheus 镜像
FROM prom/prometheus:${PROMETHEUS_VERSION}

# 标签信息
LABEL maintainer="AI-Infra-Matrix Team"
LABEL description="Prometheus Server for AI-Infra-Matrix Monitoring"
LABEL version="${PROMETHEUS_VERSION}"

# 复制配置文件
COPY prometheus.yml /etc/prometheus/prometheus.yml
COPY rules/ /etc/prometheus/rules/

# 设置数据目录权限
USER root
RUN mkdir -p /prometheus && chown -R nobody:nobody /prometheus
USER nobody

# 暴露端口
EXPOSE 9090

# 设置入口点
ENTRYPOINT [ "/bin/prometheus" ]
CMD [ "--config.file=/etc/prometheus/prometheus.yml", \
      "--storage.tsdb.path=/prometheus", \
      "--storage.tsdb.retention.time=30d", \
      "--web.enable-lifecycle", \
      "--web.enable-admin-api", \
      "--web.console.libraries=/usr/share/prometheus/console_libraries", \
      "--web.console.templates=/usr/share/prometheus/consoles" ]
