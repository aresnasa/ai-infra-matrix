#!/bin/bash

# 数据库备份和恢复脚本
# 用于Ansible Playbook Generator项目

# 设置变量
DB_HOST="localhost"
DB_PORT="5433"
DB_NAME="ansible_playbook_generator"
DB_USER="postgres"
DB_PASSWORD="postgres"
DOCKER_CONTAINER="ansible-postgres"
BACKUP_DIR="./migrations"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# 颜色输出
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

# 检查Docker容器是否运行
check_container() {
    if ! docker ps | grep -q "$DOCKER_CONTAINER"; then
        log_error "Docker容器 $DOCKER_CONTAINER 未运行"
        exit 1
    fi
    log_info "Docker容器 $DOCKER_CONTAINER 正在运行"
}

# 备份数据库结构
backup_schema() {
    log_info "开始备份数据库结构..."
    docker exec -it "$DOCKER_CONTAINER" pg_dump \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --schema-only \
        --no-owner \
        --no-privileges \
        -f "/tmp/schema_backup_${TIMESTAMP}.sql"
    
    docker cp "$DOCKER_CONTAINER:/tmp/schema_backup_${TIMESTAMP}.sql" "$BACKUP_DIR/"
    log_info "数据库结构备份完成: $BACKUP_DIR/schema_backup_${TIMESTAMP}.sql"
}

# 备份RBAC数据
backup_rbac_data() {
    log_info "开始备份RBAC数据..."
    docker exec -it "$DOCKER_CONTAINER" pg_dump \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --data-only \
        --table=permissions \
        --table=roles \
        --table=role_permissions \
        --table=users \
        --table=user_roles \
        -f "/tmp/rbac_data_backup_${TIMESTAMP}.sql"
    
    docker cp "$DOCKER_CONTAINER:/tmp/rbac_data_backup_${TIMESTAMP}.sql" "$BACKUP_DIR/"
    log_info "RBAC数据备份完成: $BACKUP_DIR/rbac_data_backup_${TIMESTAMP}.sql"
}

# 备份完整数据库
backup_full() {
    log_info "开始备份完整数据库..."
    docker exec -it "$DOCKER_CONTAINER" pg_dump \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        --no-owner \
        --no-privileges \
        -f "/tmp/full_backup_${TIMESTAMP}.sql"
    
    docker cp "$DOCKER_CONTAINER:/tmp/full_backup_${TIMESTAMP}.sql" "$BACKUP_DIR/"
    log_info "完整数据库备份完成: $BACKUP_DIR/full_backup_${TIMESTAMP}.sql"
}

# 初始化数据库
init_database() {
    log_info "开始初始化数据库..."
    if [ ! -f "$BACKUP_DIR/init_database.sql" ]; then
        log_error "初始化SQL文件不存在: $BACKUP_DIR/init_database.sql"
        exit 1
    fi
    
    # 复制初始化文件到容器
    docker cp "$BACKUP_DIR/init_database.sql" "$DOCKER_CONTAINER:/tmp/"
    
    # 执行初始化
    docker exec -it "$DOCKER_CONTAINER" psql \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -f "/tmp/init_database.sql"
    
    log_info "数据库初始化完成"
}

# 恢复数据库结构
restore_schema() {
    local backup_file="$1"
    if [ -z "$backup_file" ]; then
        log_error "请指定备份文件"
        exit 1
    fi
    
    if [ ! -f "$backup_file" ]; then
        log_error "备份文件不存在: $backup_file"
        exit 1
    fi
    
    log_info "开始恢复数据库结构..."
    
    # 复制备份文件到容器
    docker cp "$backup_file" "$DOCKER_CONTAINER:/tmp/restore_schema.sql"
    
    # 执行恢复
    docker exec -it "$DOCKER_CONTAINER" psql \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -f "/tmp/restore_schema.sql"
    
    log_info "数据库结构恢复完成"
}

# 恢复数据
restore_data() {
    local backup_file="$1"
    if [ -z "$backup_file" ]; then
        log_error "请指定备份文件"
        exit 1
    fi
    
    if [ ! -f "$backup_file" ]; then
        log_error "备份文件不存在: $backup_file"
        exit 1
    fi
    
    log_info "开始恢复数据..."
    
    # 复制备份文件到容器
    docker cp "$backup_file" "$DOCKER_CONTAINER:/tmp/restore_data.sql"
    
    # 执行恢复
    docker exec -it "$DOCKER_CONTAINER" psql \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -f "/tmp/restore_data.sql"
    
    log_info "数据恢复完成"
}

# 重置数据库
reset_database() {
    log_warn "警告: 这将删除所有数据并重新初始化数据库"
    read -p "确定要继续吗? (y/N): " confirm
    
    if [[ $confirm =~ ^[Yy]$ ]]; then
        log_info "开始重置数据库..."
        
        # 删除数据库
        docker exec -it "$DOCKER_CONTAINER" psql \
            -U "$DB_USER" \
            -d "postgres" \
            -c "DROP DATABASE IF EXISTS $DB_NAME;"
        
        # 重新创建数据库
        docker exec -it "$DOCKER_CONTAINER" psql \
            -U "$DB_USER" \
            -d "postgres" \
            -c "CREATE DATABASE $DB_NAME;"
        
        # 初始化数据库
        init_database
        
        log_info "数据库重置完成"
    else
        log_info "操作已取消"
    fi
}

# 显示帮助信息
show_help() {
    echo "数据库备份和恢复工具"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  backup-schema          备份数据库结构"
    echo "  backup-rbac           备份RBAC数据"
    echo "  backup-full           备份完整数据库"
    echo "  init                  初始化数据库"
    echo "  restore-schema FILE   恢复数据库结构"
    echo "  restore-data FILE     恢复数据"
    echo "  reset                 重置数据库(危险操作)"
    echo "  help                  显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 backup-schema                              # 备份数据库结构"
    echo "  $0 backup-rbac                               # 备份RBAC数据"
    echo "  $0 backup-full                               # 备份完整数据库"
    echo "  $0 init                                      # 初始化数据库"
    echo "  $0 restore-schema migrations/schema_backup.sql  # 恢复数据库结构"
    echo "  $0 restore-data migrations/rbac_data_backup.sql # 恢复RBAC数据"
}

# 主程序
main() {
    # 检查备份目录
    mkdir -p "$BACKUP_DIR"
    
    case "$1" in
        "backup-schema")
            check_container
            backup_schema
            ;;
        "backup-rbac")
            check_container
            backup_rbac_data
            ;;
        "backup-full")
            check_container
            backup_full
            ;;
        "init")
            check_container
            init_database
            ;;
        "restore-schema")
            check_container
            restore_schema "$2"
            ;;
        "restore-data")
            check_container
            restore_data "$2"
            ;;
        "reset")
            check_container
            reset_database
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            log_error "未知选项: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 运行主程序
main "$@"
