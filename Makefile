# Hugo 博客构建脚本
#
# 主要 target：
#   make build  - 编译 uploader → 同步图片到 OSS → hugo 构建（默认）
#   make dev    - 本地开发：hugo server --disableFastRender，不触发上传
#   make clean  - 清理上传工具二进制、映射文件、构建产物
#   make help   - 显示帮助信息
#
# 所有环境变量均允许外部覆盖；未设置时使用下方 `?=` 给出的默认值。

# ---------- 可配置变量 ----------
# OSS 地域，例如 cn-hangzhou、cn-beijing、oss-cn-shanghai
OSS_REGION           ?= cn-shenzhen
# OSS bucket 名称（必填）
OSS_BUCKET_NAME      ?= blog-cetkit
# OSS 存储前缀
OSS_IMAGE_PREFIX     ?= blog-images/
# CDN 自定义域名；留空则使用 OSS 默认域名
OSS_CUSTOM_DOMAIN    ?=
# Hugo content 根目录
HUGO_CONTENT_DIR     ?= ./content
# Hugo data 目录
HUGO_DATA_DIR        ?= ./data
# 额外图片搜索根（逗号分隔），适用于 static/ox-hugo 等场景
OSS_EXTRA_SEARCH_DIRS ?=

# Go 上传工具源码所在目录
UPLOADER_DIR         := scripts/oss-uploader
# Go 上传工具编译产物路径
UPLOADER_BIN         := $(UPLOADER_DIR)/oss-uploader
# Go 上传工具源码
UPLOADER_SRC         := $(UPLOADER_DIR)/main.go
# 图片映射文件输出路径
IMAGE_MAP            := $(HUGO_DATA_DIR)/oss-images.json

# ---------- 默认目标 ----------
.PHONY: build
build: $(UPLOADER_BIN) $(IMAGE_MAP)
	@echo "🚀 启动 Hugo 构建"
	@hugo --minify
	@echo "✅ Hugo 构建完成，产物位于 public/"

# 编译 Go 上传工具：源码变更时才重新编译
$(UPLOADER_BIN): $(UPLOADER_SRC) $(UPLOADER_DIR)/go.mod $(UPLOADER_DIR)/go.sum
	@echo "🔨 编译 OSS 上传工具"
	@cd $(UPLOADER_DIR) && CGO_ENABLED=0 go build -o oss-uploader .

# 执行上传工具，生成/更新图片映射
$(IMAGE_MAP): $(UPLOADER_BIN)
	@echo "📤 上传本地图片到 OSS"
	@OSS_REGION='$(OSS_REGION)' \
	 OSS_BUCKET_NAME='$(OSS_BUCKET_NAME)' \
	 OSS_IMAGE_PREFIX='$(OSS_IMAGE_PREFIX)' \
	 OSS_CUSTOM_DOMAIN='$(OSS_CUSTOM_DOMAIN)' \
	 HUGO_CONTENT_DIR='$(HUGO_CONTENT_DIR)' \
	 HUGO_DATA_DIR='$(HUGO_DATA_DIR)' \
	 OSS_EXTRA_SEARCH_DIRS='$(OSS_EXTRA_SEARCH_DIRS)' \
	 ./$(UPLOADER_BIN)

# ---------- 本地开发 ----------
.PHONY: dev
dev:
	@echo "🛠  启动 Hugo 开发服务器（不触发 OSS 上传）"
	@hugo server --disableFastRender

# ---------- 清理 ----------
.PHONY: clean
clean:
	@echo "🧹 清理构建产物"
	@rm -f $(UPLOADER_BIN)
	@rm -f $(IMAGE_MAP)
	@rm -rf public/
	@rm -rf resources/
	@echo "✅ 清理完成"

# ---------- 帮助 ----------
.PHONY: help
help:
	@echo "可用命令："
	@echo "  make build  - 编译 uploader → 同步图片到 OSS → hugo 构建（默认）"
	@echo "  make dev    - 本地开发：hugo server --disableFastRender，不触发上传"
	@echo "  make clean  - 清理上传工具二进制、映射文件、构建产物"
	@echo "  make help   - 显示本帮助信息"
	@echo ""
	@echo "环境变量（均可外部覆盖）："
	@echo "  OSS_REGION         默认 cn-hangzhou"
	@echo "  OSS_BUCKET_NAME    必填"
	@echo "  OSS_IMAGE_PREFIX   默认 blog-images/"
	@echo "  OSS_CUSTOM_DOMAIN  留空使用 OSS 默认域名"
	@echo "  HUGO_CONTENT_DIR   默认 ./content"
	@echo "  HUGO_DATA_DIR      默认 ./data"
	@echo "  OSS_EXTRA_SEARCH_DIRS  默认空，逗号分隔，例如 static/ox-hugo,static/images"
