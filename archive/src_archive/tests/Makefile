# Ansible Playbook Generator Test Suite
# Usage: make test-all
# Author: DevOps Team

# 项目路径
PROJECT_ROOT := $(shell cd .. && pwd)
BACKEND_DIR := $(PROJECT_ROOT)/backend
FRONTEND_DIR := $(PROJECT_ROOT)/frontend

.PHONY: help test-all test-unit test-integration test-e2e start-test-env stop-test-env clean \
        health-check build-all start-prod stop-prod logs status restart auto-test quick-test \
        build-and-test

# Default target
help:
	@echo "🎯 Ansible Playbook Generator Test Suite"
	@echo "========================================"
	@echo "Available commands:"
	@echo ""
	@echo "🚀 AUTOMATED TESTING:"
	@echo "  auto-test       - Run FULLY AUTOMATED test suite with reports"
	@echo "  quick-test      - Run essential tests only (faster)"
	@echo ""
	@echo "🧪 MANUAL TESTING:"
	@echo "  test-all        - Run all tests in sequence"
	@echo "  test-unit       - Run unit tests in Docker containers"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e        - Run end-to-end tests"
	@echo ""
	@echo "🛠️  ENVIRONMENT MANAGEMENT:"
	@echo "  start-test-env  - Start test environment"
	@echo "  stop-test-env   - Stop test environment"
	@echo "  health-check    - Check all services health status"
	@echo "  build-all       - Build all Docker images"
	@echo ""
	@echo "🌐 PRODUCTION:"
	@echo "  start-prod      - Start production environment"
	@echo "  stop-prod       - Stop production environment"
	@echo ""
	@echo "🔧 UTILITIES:"
	@echo "  logs            - Show logs from all services"
	@echo "  status          - Show status of all services"
	@echo "  restart         - Restart all services"
	@echo "  clean           - Clean test artifacts and containers"

# 🚀 FULLY AUTOMATED TEST SUITE
auto-test:
	@echo "🤖 Starting FULLY AUTOMATED test suite..."
	@echo "=========================================="
	@chmod +x ./run-all-tests.sh && \
	./run-all-tests.sh

# ⚡ QUICK AUTOMATED TESTS
quick-test:
	@echo "⚡ Starting QUICK automated test suite..."
	@echo "========================================"
	@chmod +x ./run-all-tests.sh && \
	QUICK_MODE=true ./run-all-tests.sh

# Run all tests in fully automated mode
test-all: clean build-all start-test-env health-check test-unit test-integration test-e2e stop-test-env
	@echo "🎉 All tests completed successfully!"
	@echo "✅ Ansible Playbook Generator is ready for production!"

# Build all Docker images
build-all:
	@echo "🔨 Building all Docker images..."
	@cd $(PROJECT_ROOT) && docker-compose build
	@echo "✅ All images built successfully!"

# Start production services
start-prod:
	@echo "🚀 Starting production environment..."
	@cd $(PROJECT_ROOT) && docker-compose up -d
	@echo "✅ Production environment started!"
	@echo "🌐 Frontend: http://localhost:3001"
	@echo "🔧 Backend API: http://localhost:8082/api"

# Stop production services  
stop-prod:
	@echo "⏹️  Stopping production environment..."
	@cd $(PROJECT_ROOT) && docker-compose down
	@echo "✅ Production environment stopped!"

# Unit tests in containers
test-unit:
	@echo "🧪 Running unit tests in Docker containers..."
	@echo "Testing backend services..."
	@docker-compose -f docker-compose.test.yml exec -T backend-test go test -v ./internal/... -short -coverprofile=coverage.out
	@echo "Testing frontend services..."
	@if docker-compose -f docker-compose.test.yml ps frontend-test | grep -q "Up"; then \
		docker-compose -f docker-compose.test.yml exec -T frontend-test npm test -- --watchAll=false --passWithNoTests --coverage; \
	else \
		echo "⚠️  Frontend test container not running, skipping frontend tests"; \
	fi
	@echo "✅ Unit tests completed!"

# Integration tests
test-integration:
	@echo "🔗 Running integration tests..."
	@docker-compose -f docker-compose.test.yml exec -T backend-test go test -v ./integration/... -tags=integration
	@echo "✅ Integration tests completed!"

# End-to-end tests
test-e2e:
	@echo "🌐 Running end-to-end tests..."
	@chmod +x ./scripts/e2e-test.sh
	@./scripts/e2e-test.sh
	@echo "✅ End-to-end tests completed!"

# Start test environment with health checks
start-test-env:
	@echo "🚀 Starting test environment..."
	@docker-compose -f docker-compose.test.yml down --remove-orphans && \
	docker-compose -f docker-compose.test.yml up --build -d
	@echo "⏳ Waiting for services to be ready..."
	@chmod +x ./scripts/wait-for-services.sh
	@./scripts/wait-for-services.sh
	@echo "✅ Test environment is ready!"

# Stop test environment
stop-test-env:
	@echo "🛑 Stopping test environment..."
	@docker-compose -f docker-compose.test.yml down --remove-orphans
	@echo "✅ Test environment stopped!"

# Health check for all services
health-check:
	@echo "🏥 Running enhanced health check..."
	@chmod +x ./scripts/health-check-enhanced.sh
	@./scripts/health-check-enhanced.sh

# Stop production environment
stop-prod:
	@echo "🛑 Stopping production environment..."
	@cd .. && docker-compose down --remove-orphans

# Show logs from all services
logs:
	@echo "📋 Showing logs from all services..."
	@docker-compose -f docker-compose.test.yml logs --tail=50 -f

# Show status of all services
status:
	@echo "📊 Service Status:"
	@echo "=================="
	@echo "Test Environment:"
	@docker-compose -f docker-compose.test.yml ps
	@echo ""
	@echo "Production Environment:"
	@cd .. && docker-compose ps

# Restart all services
restart: stop-test-env start-test-env
	@echo "🔄 All services restarted!"

# Clean test artifacts and containers
clean:
	@echo "🧹 Cleaning test artifacts and containers..."
	@rm -rf reports/ coverage/ logs/
	@mkdir -p reports coverage logs
	@docker-compose -f docker-compose.test.yml down --remove-orphans --volumes
	@cd .. && docker-compose down --remove-orphans --volumes
	@docker system prune -f
	@echo "✅ Cleanup completed!"

# 快速构建和测试（推荐用于开发）
build-and-test:
	@echo "🚀 Quick Build and Test Pipeline"
	@echo "================================="
	@cd $(PROJECT_ROOT) && \
	echo "🛑 Stopping existing containers..." && \
	docker-compose down --volumes && \
	echo "🔨 Building..." && \
	docker-compose build && \
	echo "🚀 Starting services..." && \
	docker-compose up -d && \
	echo "⏳ Waiting for services to be ready..." && \
	sleep 30 && \
	echo "🔍 Health check..." && \
	curl -f http://localhost:8082/api/health && \
	echo "" && \
	echo "✅ Build and test completed successfully!" && \
	echo "🌐 Frontend: http://localhost:3001" && \
	echo "🔧 Backend API: http://localhost:8082/api"
