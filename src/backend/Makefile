# Makefile for ansible-playbook-generator backend

.PHONY: build run init test clean

# 构建主应用
build:
	go build -o bin/backend cmd/main.go

# 构建初始化程序
build-init:
	go build -o bin/init cmd/init/main.go

# 运行主应用
run:
	go run cmd/main.go

# 运行初始化程序（本地）
init:
	go run cmd/init/main.go

# 运行初始化程序（Docker环境）
init-docker:
	docker-compose exec backend go run cmd/init/main.go

# 构建Docker镜像
docker-build:
	docker-compose build backend

# 运行测试
test:
	go test ./...

# 清理构建文件
clean:
	rm -rf bin/
	rm -rf outputs/
	rm -rf uploads/

# 格式化代码
fmt:
	go fmt ./...

# 检查代码
vet:
	go vet ./...

# 安装依赖
deps:
	go mod download
	go mod tidy

# 重置数据库并初始化
reset-all:
	docker-compose down -v
	docker-compose up -d postgres redis openldap
	sleep 10
	docker-compose up -d backend
	sleep 5
	$(MAKE) init-docker

# 查看日志
logs:
	docker-compose logs -f backend

# 查看LDAP日志
logs-ldap:
	docker-compose logs -f openldap

# 帮助
help:
	@echo "Available commands:"
	@echo "  build        - Build the main application"
	@echo "  build-init   - Build the initialization program"
	@echo "  run          - Run the main application locally"
	@echo "  init         - Run initialization program locally"
	@echo "  init-docker  - Run initialization program in Docker"
	@echo "  docker-build - Build Docker image"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build files"
	@echo "  fmt          - Format code"
	@echo "  vet          - Check code"
	@echo "  deps         - Download dependencies"
	@echo "  reset-all    - Reset everything and reinitialize"
	@echo "  logs         - View backend logs"
	@echo "  logs-ldap    - View LDAP logs"
	@echo "  help         - Show this help"
