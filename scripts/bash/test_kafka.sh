#!/bin/bash

# Kafka 测试脚本
# 用于验证Kafka服务的功能和连接性

set -e

echo "🚀 开始Kafka服务测试..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker环境
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或不在PATH中"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装或不在PATH中"
        exit 1
    fi

    log_info "Docker 环境检查通过"
}

# 等待服务启动
wait_for_service() {
    local service_name=$1
    local host=$2
    local port=$3
    local max_attempts=30
    local attempt=1

    log_info "等待 $service_name 服务启动..."

    while [ $attempt -le $max_attempts ]; do
        if docker-compose exec -T $service_name nc -z $host $port 2>/dev/null; then
            log_info "$service_name 服务已就绪"
            return 0
        fi

        log_warn "等待 $service_name 服务... (尝试 $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done

    log_error "$service_name 服务启动超时"
    return 1
}

# 测试Zookeeper连接
test_zookeeper() {
    log_info "测试Zookeeper连接..."

    if ! docker-compose exec -T zookeeper bash -c "echo 'ruok' | nc localhost 2181" | grep -q "imok"; then
        log_error "Zookeeper 连接测试失败"
        return 1
    fi

    log_info "Zookeeper 连接测试通过"
    return 0
}

# 测试Kafka连接
test_kafka() {
    log_info "测试Kafka连接..."

    # 等待Kafka启动
    sleep 10

    # 创建测试主题
    if ! docker-compose exec -T kafka kafka-topics --create --topic test-topic --partitions 1 --replication-factor 1 --bootstrap-server localhost:9092 2>/dev/null; then
        log_error "创建测试主题失败"
        return 1
    fi

    # 发送测试消息
    if ! echo "Hello Kafka" | docker-compose exec -T kafka kafka-console-producer --topic test-topic --bootstrap-server localhost:9092 2>/dev/null; then
        log_error "发送测试消息失败"
        return 1
    fi

    # 消费测试消息
    if ! docker-compose exec -T kafka kafka-console-consumer --topic test-topic --from-beginning --max-messages 1 --bootstrap-server localhost:9092 2>/dev/null | grep -q "Hello Kafka"; then
        log_error "消费测试消息失败"
        return 1
    fi

    # 清理测试主题
    docker-compose exec -T kafka kafka-topics --delete --topic test-topic --bootstrap-server localhost:9092 2>/dev/null || true

    log_info "Kafka 连接测试通过"
    return 0
}

# 测试Kafka主题
test_kafka_topics() {
    log_info "测试Kafka主题创建..."

    local topics=("ai-chat-messages" "ai-message-events" "ai-message-cache")

    for topic in "${topics[@]}"; do
        if ! docker-compose exec -T kafka kafka-topics --describe --topic $topic --bootstrap-server localhost:9092 2>/dev/null; then
            log_error "主题 $topic 不存在，尝试创建..."
            if ! docker-compose exec -T kafka kafka-topics --create --topic $topic --partitions 3 --replication-factor 1 --bootstrap-server localhost:9092 2>/dev/null; then
                log_error "创建主题 $topic 失败"
                return 1
            fi
        fi
        log_info "主题 $topic 检查通过"
    done

    return 0
}

# 测试Kafka UI
test_kafka_ui() {
    log_info "测试Kafka UI..."

    if ! curl -f http://localhost:9095 2>/dev/null; then
        log_warn "Kafka UI 可能未启动或不可访问"
        return 1
    fi

    log_info "Kafka UI 测试通过"
    return 0
}

# 主测试函数
main() {
    log_info "开始Kafka服务完整测试"

    # 检查Docker环境
    check_docker

    # 等待Zookeeper启动
    if ! wait_for_service "zookeeper" "localhost" "2181"; then
        exit 1
    fi

    # 等待Kafka启动
    if ! wait_for_service "kafka" "localhost" "9092"; then
        exit 1
    fi

    # 测试Zookeeper
    if ! test_zookeeper; then
        exit 1
    fi

    # 测试Kafka
    if ! test_kafka; then
        exit 1
    fi

    # 测试Kafka主题
    if ! test_kafka_topics; then
        exit 1
    fi

    # 测试Kafka UI（可选）
    test_kafka_ui || true

    log_info "🎉 所有Kafka测试通过！"
    log_info ""
    log_info "Kafka服务信息:"
    log_info "  - Zookeeper: localhost:2181"
    log_info "  - Kafka: localhost:9092 (内部), localhost:9094 (外部)"
    log_info "  - Kafka UI: http://localhost:9095"
    log_info ""
    log_info "可用主题:"
    log_info "  - ai-chat-messages: AI聊天消息"
    log_info "  - ai-message-events: 消息事件"
    log_info "  - ai-message-cache: 消息缓存"
}

# 如果脚本被直接执行
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
