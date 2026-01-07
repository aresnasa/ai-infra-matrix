# SafeLine WAF - External Docker Images
# 雷池 WAF 使用预构建的 Docker 镜像，无需本地构建
# 官方文档: https://github.com/chaitin/SafeLine
#
# 注意: SafeLine 包含多个镜像，由 SAFELINE_IMAGES 数组在 build.sh 中管理
# 此文件仅用于 discover_services 识别组件类型

{{SAFELINE_IMAGE_PREFIX}}/safeline-tengine{{SAFELINE_REGION}}{{SAFELINE_ARCH_SUFFIX}}:{{SAFELINE_IMAGE_TAG}}
