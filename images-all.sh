# 构建所有服务（多架构）
MULTI_ARCH_BUILD=true ./build.sh build-all

# 指定特定平台
MULTI_ARCH_BUILD=true TARGET_PLATFORMS=linux/amd64,linux/arm64 ./build.sh build-all

# 从 Harbor 拉取多架构依赖镜像
MULTI_ARCH_BUILD=true ./build.sh harbor-pull-deps registry.example.com/ai-infra

# 拉取所有镜像（服务+依赖）
MULTI_ARCH_BUILD=true ./build.sh harbor-pull-all registry.example.com/ai-infra v0.3.8

# 推送所有服务（包含多架构 manifest）
MULTI_ARCH_BUILD=true ./build.sh push-all registry.example.com/ai-infra v0.3.8

# 推送单个服务
MULTI_ARCH_BUILD=true ./build.sh push backend v0.3.8 registry.example.com/ai-infra