# Salt State: 节点指标采集
# 部署并配置定期采集 GPU 驱动版本、IB 状态等信息的脚本

{% set node_metrics = pillar.get('node_metrics', {}) %}
{% set callback_url = node_metrics.get('callback_url', 'http://ai-infra-matrix:8080/api/saltstack/node-metrics/callback') %}
{% set api_token = node_metrics.get('api_token', '') %}
{% set collect_interval = node_metrics.get('collect_interval', 3) %}

# 确保采集脚本目录存在
/opt/ai-infra/metrics:
  file.directory:
    - makedirs: True
    - mode: 755

# 部署采集脚本
/opt/ai-infra/metrics/collect_node_metrics.sh:
  file.managed:
    - source: salt://node-metrics/files/collect_node_metrics.sh
    - mode: 755
    - require:
      - file: /opt/ai-infra/metrics

# 部署回调配置
/opt/ai-infra/metrics/callback.conf:
  file.managed:
    - source: salt://node-metrics/files/callback.conf
    - mode: 600
    - template: jinja
    - defaults:
        callback_url: {{ callback_url }}
        api_token: {{ api_token }}
    - require:
      - file: /opt/ai-infra/metrics

# 创建定时任务（根据配置的间隔执行）
node_metrics_cron:
  cron.present:
    - name: /opt/ai-infra/metrics/collect_node_metrics.sh >> /var/log/ai-infra-metrics.log 2>&1
    - user: root
    - minute: '*/{{ collect_interval }}'
    - require:
      - file: /opt/ai-infra/metrics/collect_node_metrics.sh
      - file: /opt/ai-infra/metrics/callback.conf

# 首次执行采集（部署后立即执行）
run_initial_collection:
  cmd.run:
    - name: /opt/ai-infra/metrics/collect_node_metrics.sh >> /var/log/ai-infra-metrics.log 2>&1 &
    - bg: True
    - require:
      - file: /opt/ai-infra/metrics/collect_node_metrics.sh
      - file: /opt/ai-infra/metrics/callback.conf
