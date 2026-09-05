# Hugo 博客构建脚本
#
# 主要 target：
#   make build       - 本地：编译 uploader → 同步图片到 OSS → hugo 构建（默认）
#   make oss-replace - CI：列举 OSS 已有图片，把 md 里的本地链接替换为 OSS URL
#   make build-ci    - CI：只跑 hugo 构建，不做任何 OSS 读写
#   make dev         - 本地开发：hugo server --disableFastRender，不触发上传
#   make clean       - 清理两个工具的二进制、映射文件、构建产物
#   make help        - 显示帮助信息
#
# 本地与 CI 走两条不同的路，原因是 static/ 目录未纳入 git：
#   本地有真实图片文件 → make build（上传 + 渲染期替换，需要 data/oss-images.json）
#   CI 检出后没有图片 → make oss-replace（列举 OSS 反查 URL 改写 md）+ make build-ci
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
# 追加给 hugo 命令的额外参数（默认空）。
# 本地留空即可，使用默认的 hugo.yaml。
# CI 里通过它注入叠加配置：
#   make build-ci HUGO_EXTRA_ARGS='--config "hugo.yaml,.github/hugo-ci.yaml"'
# 之所以要做成变量而不是直接写死 --themesDir：Hugo 0.165 实测命令行 --themesDir
# 会被 hugo.yaml 里的 themesDir 覆盖回去，只能通过追加配置文件的方式覆盖。
HUGO_EXTRA_ARGS       ?=
# 严格模式开关（1/true/yes 生效）：oss-replace 遇到「md 里有引用但 OSS 上找不到」
# 时不再只是告警，而是以非零状态码退出，让 CI 直接失败。
# 用法：make oss-replace STRICT=1
STRICT                ?= 0

# Go 上传工具源码所在目录
UPLOADER_DIR         := scripts/oss-uploader
# Go 上传工具编译产物路径
UPLOADER_BIN         := $(UPLOADER_DIR)/oss-uploader
# Go 上传工具源码
UPLOADER_SRC         := $(UPLOADER_DIR)/main.go
# 图片映射文件输出路径
IMAGE_MAP            := $(HUGO_DATA_DIR)/oss-images.json

# Go 链接替换工具（CI 用）源码所在目录
REPLACER_DIR         := scripts/oss-replacer
# Go 链接替换工具编译产物路径
REPLACER_BIN         := $(REPLACER_DIR)/oss-replacer
# Go 链接替换工具源码
REPLACER_SRC         := $(REPLACER_DIR)/main.go

# ---------- 默认目标（本地用）----------
# 本地完整流程：编译上传工具 → 把 content 里引用的本地图片传到 OSS → hugo 构建。
# 与 CI 的区别：本地有 static/ 下的真实图片文件，所以能算 MD5 做增量上传，
# 并由 Hugo 的 render hook 读 data/oss-images.json 完成渲染期替换。
.PHONY: build
build: $(UPLOADER_BIN) $(IMAGE_MAP)
	@echo "🚀 启动 Hugo 构建"
	@hugo $(HUGO_EXTRA_ARGS) --minify
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

# ---------- CI 构建 ----------
# 为什么 CI 不复用 build：static/ 目录没有纳入 git，CI 检出后根本没有图片文件，
# 上传工具找不到任何可传的东西，data/oss-images.json 也生成不出来。
# 所以 CI 走另一条路——不上传，只列举 OSS 上已有的对象，按文件名反查 URL，
# 直接改写工作区里 md 的本地图片链接，然后构建。
#
# 前提：本地必须先跑过 make build 把图片传上去，否则 CI 查不到对应对象
#（默认只告警，STRICT=1 时构建失败）。

# 编译 Go 链接替换工具：源码变更时才重新编译
$(REPLACER_BIN): $(REPLACER_SRC) $(REPLACER_DIR)/go.mod $(REPLACER_DIR)/go.sum
	@echo "🔨 编译 OSS 链接替换工具"
	@cd $(REPLACER_DIR) && CGO_ENABLED=0 go build -o oss-replacer .

# 执行链接替换工具。
# 注意：这里刻意不依赖 $(IMAGE_MAP)——CI 里不需要那份映射文件。
# ⚠️ 本步骤会真实改写 content/ 下的 md 文件。CI 里改的是临时检出的副本，无所谓；
#    在本地跑完记得用 git checkout -- content/ 还原，别把替换结果提交进去。
.PHONY: oss-replace
oss-replace: $(REPLACER_BIN)
	@echo "🔗 把 md 中的本地图片链接替换为 OSS URL"
	@OSS_REGION='$(OSS_REGION)' \
	 OSS_BUCKET_NAME='$(OSS_BUCKET_NAME)' \
	 OSS_IMAGE_PREFIX='$(OSS_IMAGE_PREFIX)' \
	 OSS_CUSTOM_DOMAIN='$(OSS_CUSTOM_DOMAIN)' \
	 HUGO_CONTENT_DIR='$(HUGO_CONTENT_DIR)' \
	 STRICT='$(STRICT)' \
	 ./$(REPLACER_BIN)

# CI 的构建步骤：只跑 hugo，不做任何 OSS 相关的读写。
# 与 oss-replace 拆成两个 target，是为了让 OSS 凭证只出现在替换那一步，
# 构建步骤完全拿不到 AccessKey。
.PHONY: build-ci
build-ci:
	@echo "🚀 启动 Hugo 构建（CI）"
	@hugo $(HUGO_EXTRA_ARGS) --minify
	@echo "✅ Hugo 构建完成，产物位于 public/"

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
	@rm -f $(REPLACER_BIN)
	@rm -f $(IMAGE_MAP)
	@rm -rf public/
	@rm -rf resources/
	@echo "✅ 清理完成"

# ---------- 帮助 ----------
.PHONY: help
help:
	@echo "可用命令："
	@echo "  make build       - 本地：编译 uploader → 同步图片到 OSS → hugo 构建（默认）"
	@echo "  make oss-replace - CI：列举 OSS 已有图片，把 md 里的本地链接换成 OSS URL"
	@echo "  make build-ci    - CI：只跑 hugo 构建，不做任何 OSS 读写"
	@echo "  make dev         - 本地开发：hugo server --disableFastRender，不触发上传"
	@echo "  make clean       - 清理两个工具的二进制、映射文件、构建产物"
	@echo "  make help        - 显示本帮助信息"
	@echo ""
	@echo "本地与 CI 的分工："
	@echo "  本地有 static/ 下的真实图片，走 make build（上传 + 渲染期替换）；"
	@echo "  static/ 未纳入 git，CI 检出后没有图片，改走 oss-replace + build-ci。"
	@echo "  ⚠️ 因此新增/改动图片后必须先在本地跑 make build 上传，再推代码，"
	@echo "     否则 CI 查不到对应对象（默认只告警，STRICT=1 时构建失败）。"
	@echo "  ⚠️ make oss-replace 会真实改写 content/ 下的 md，本地跑完记得"
	@echo "     git checkout -- content/ 还原，不要把替换结果提交进仓库。"
	@echo ""
	@echo "环境变量（均可外部覆盖）："
	@echo "  OSS_REGION         默认 cn-hangzhou"
	@echo "  OSS_BUCKET_NAME    必填"
	@echo "  OSS_IMAGE_PREFIX   默认 blog-images/"
	@echo "  OSS_CUSTOM_DOMAIN  留空使用 OSS 默认域名"
	@echo "  HUGO_CONTENT_DIR   默认 ./content"
	@echo "  HUGO_DATA_DIR      默认 ./data"
	@echo "  OSS_EXTRA_SEARCH_DIRS  默认空，逗号分隔，例如 static/ox-hugo,static/images（仅 build 用）"
	@echo "  HUGO_EXTRA_ARGS        默认空，追加给 hugo 的参数，CI 用于指定叠加配置"
	@echo "  STRICT                 默认 0；设为 1 时 oss-replace 遇到缺失图片即失败"
