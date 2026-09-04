// Package main 实现 Hugo content 目录的图片自动上传到阿里云 OSS，
// 并生成 data/oss-images.json 映射文件供 Hugo render-image.html 使用。
//
// Markdown 源文件不会被修改；图片路径的替换发生在 Hugo 渲染阶段。
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 工具日志 emoji 前缀
const (
	emojiOK    = "✅"
	emojiInfo  = "ℹ️ "
	emojiWarn  = "⚠️ "
	emojiError = "❌"
	emojiUp    = "⬆️ "
	emojiSkip  = "⏭️ "
)

// 配置文件条目：本地路径 → OSS key / 远程 URL
type ImageEntry struct {
	OSSKey string `json:"oss_key"`
	URL    string `json:"url"`
}

// 整体映射：键为图片在磁盘上的绝对路径
type ImageMap map[string]ImageEntry

// 从环境变量读取并组装配置
type Config struct {
	Region          string   // OSS 地域
	Bucket          string   // OSS bucket
	Prefix          string   // OSS 存储前缀
	CustomDomain    string   // CDN 自定义域名（可选）
	ContentDir      string   // Hugo content 根目录
	StaticDirs      []string // 额外搜索目录（用于 static/ 等场景）
	DataDir         string   // Hugo data 目录
}

// 配置默认值
func defaultConfig() Config {
	return Config{
		Region:     "cn-hangzhou",
		Prefix:     "blog-images/",
		ContentDir: "./content",
		StaticDirs: []string{},
		DataDir:    "./data",
	}
}

// 加载环境变量，未填的走默认；必填字段缺失则报错退出。
func loadConfig() (Config, error) {
	c := defaultConfig()
	if v := os.Getenv("OSS_REGION"); v != "" {
		c.Region = v
	}
	if v := os.Getenv("OSS_BUCKET_NAME"); v != "" {
		c.Bucket = v
	}
	if v := os.Getenv("OSS_IMAGE_PREFIX"); v != "" {
		c.Prefix = v
	}
	if v := os.Getenv("OSS_CUSTOM_DOMAIN"); v != "" {
		c.CustomDomain = v
	}
	if v := os.Getenv("HUGO_CONTENT_DIR"); v != "" {
		c.ContentDir = v
	}
	if v := os.Getenv("HUGO_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("OSS_EXTRA_SEARCH_DIRS"); v != "" {
		// 多个目录用逗号分隔，例如 "static/ox-hugo,static/images"
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				c.StaticDirs = append(c.StaticDirs, p)
			}
		}
	}
	if c.Bucket == "" {
		return c, fmt.Errorf("OSS_BUCKET_NAME 环境变量未设置")
	}
	if !strings.HasSuffix(c.Prefix, "/") {
		c.Prefix += "/"
	}
	return c, nil
}

// 匹配 Markdown 图片语法：![alt](path "title")
//  - alt 可为空
//  - path 不可包含空白或右括号
//  - title 可选，由英文双引号包裹
var imageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)`)

// 匹配指向本地图片的 Markdown 链接：[text](path)，用于渲染时生成 <a href>
// 仅匹配目标为图片扩展名的链接，避免误处理站内导航链接
var imageLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+\.(?:png|jpg|jpeg|gif|svg|webp|bmp|ico))(?:\s+"([^"]*)")?\)`)

// 匹配 Hugo figure / img shortcode 中的 src 属性
//   {{< figure src="/ox-hugo/foo.png" >}}
//   {{< img src="images/foo.png" >}}
// 支持双引号或单引号包裹的 src 值
var figureSrcRegex = regexp.MustCompile(`\{\{<\s*(?:figure|img)\s+[^}>]*?src=["']([^"']+)["']`)

// 支持的图片扩展名（小写）
var supportedExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".svg": true, ".webp": true,
	".bmp": true, ".ico": true,
}

// 判断路径是否为远程 URL（http/https/data:），若是则跳过上传
func isRemoteURL(p string) bool {
	lp := strings.ToLower(p)
	return strings.HasPrefix(lp, "http://") ||
		strings.HasPrefix(lp, "https://") ||
		strings.HasPrefix(lp, "data:")
}

// 在 markdown 文本里提取所有本地图片引用，返回解析后的绝对路径列表。
// 同时识别 Markdown 图片语法 `![alt](path)` 和 Hugo figure/img shortcode
// `{{< figure src="path" >}}`。
func extractLocalImages(content string, mdAbs string, cfg Config) []string {
	var found []string
	add := func(path string) {
		if isRemoteURL(path) {
			return
		}
		abs, err := resolveImagePath(path, mdAbs, cfg)
		if err != nil {
			fmt.Printf("%s 解析图片路径失败 %q（位于 %s）：%v\n", emojiWarn, path, mdAbs, err)
			return
		}
		found = append(found, abs)
	}

	for _, m := range imageRegex.FindAllStringSubmatch(content, -1) {
		add(m[2])
	}
	// 同步匹配 [text](image) 链接语法，确保 <a href> 也能命中 OSS URL
	for _, m := range imageLinkRegex.FindAllStringSubmatch(content, -1) {
		add(m[2])
	}
	for _, m := range figureSrcRegex.FindAllStringSubmatch(content, -1) {
		add(m[1])
	}
	return found
}

// 把 markdown 中引用的图片路径解析为磁盘上的绝对路径。
//
// 查找顺序（找到第一个存在的文件即返回）：
//  1. 规范路径：
//     - 以 '/' 开头（如 /ox-hugo/x.png）→ 相对 Hugo content 根目录
//     - 其他（images/x.png、x.png）     → 相对当前 .md 文件所在目录
//  2. static/ 镜像（与规范路径相同的相对结构，根从 content 换成 static）
//  3. OSS_EXTRA_SEARCH_DIRS 配置的额外目录 + basename（适用于 ox-hugo 这种
//     把所有图片集中放在 static/ox-hugo/ 的场景，即使 markdown 引用是
//     裸文件名或 images/ 子目录形式）
func resolveImagePath(ref string, mdAbs string, cfg Config) (string, error) {
	// 先按扩展名过滤，避免误把无关文件当图片
	ext := strings.ToLower(filepath.Ext(ref))
	if !supportedExts[ext] {
		return "", fmt.Errorf("不支持的图片扩展名 %q", ext)
	}

	contentAbs, err := filepath.Abs(cfg.ContentDir)
	if err != nil {
		return "", err
	}

	// 计算候选路径列表
	candidates := buildCandidates(ref, mdAbs, contentAbs, cfg.StaticDirs)

	// 汇总可信根目录（content + static 镜像根 + 用户配置的额外搜索根）
	trustedRoots := []string{contentAbs}
	staticRoot := filepath.Join(filepath.Dir(contentAbs), "static")
	trustedRoots = append(trustedRoots, staticRoot)
	for _, dir := range cfg.StaticDirs {
		if abs, err := filepath.Abs(dir); err == nil {
			trustedRoots = append(trustedRoots, abs)
		}
	}

	// 依次尝试，第一个存在且在可信根目录内的即为结果
	for _, c := range candidates {
		if c == "" {
			continue
		}
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if !isUnderTrustedRoots(abs, trustedRoots) {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}
	return "", fmt.Errorf("在所有候选路径中未找到 %q", ref)
}

// 判断 abs 路径是否位于任一可信根目录下。
func isUnderTrustedRoots(abs string, roots []string) bool {
	for _, root := range roots {
		if abs == root {
			return true
		}
		if strings.HasPrefix(abs, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// 根据引用和上下文生成候选绝对路径列表。
// ref 是 markdown 中的图片路径；mdAbs 是 .md 文件的绝对路径；
// contentAbs 是 content 根目录绝对路径；extras 是用户配置的额外搜索根。
func buildCandidates(ref, mdAbs, contentAbs string, extras []string) []string {
	trimmed := strings.TrimPrefix(ref, "/")
	base := filepath.Base(ref)

	var primary string
	if strings.HasPrefix(ref, "/") {
		primary = filepath.Join(contentAbs, trimmed)
	} else {
		primary = filepath.Join(filepath.Dir(mdAbs), ref)
	}

	// 第一个候选：规范路径
	candidates := []string{primary}

	// 第二个候选：static/ 镜像（结构保持一致）
	staticMirror := filepath.Join(filepath.Dir(contentAbs), "static", trimmed)
	candidates = append(candidates, staticMirror)

	// 第三组：每个额外搜索目录 + basename
	for _, dir := range extras {
		dirAbs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		candidates = append(candidates, filepath.Join(dirAbs, base))
	}
	return candidates
}

// 计算文件 MD5 的前 12 位
func shortHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:12], nil
}

// 根据 OSS key 与可选的自定义 CDN 域名，组装最终访问 URL
func buildURL(ossKey, customDomain, bucket, region string) string {
	if customDomain != "" {
		base := strings.TrimRight(customDomain, "/")
		return base + "/" + ossKey
	}
	return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", bucket, region, ossKey)
}

// 读取已存在的映射文件；不存在或解析失败则返回空映射（视为首次运行）
func loadExisting(dataDir string) ImageMap {
	path := filepath.Join(dataDir, "oss-images.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return ImageMap{}
	}
	var m ImageMap
	if err := json.Unmarshal(b, &m); err != nil {
		fmt.Printf("%s 现有映射文件解析失败，将重建：%v\n", emojiWarn, err)
		return ImageMap{}
	}
	fmt.Printf("%s 已读取 %d 条历史映射\n", emojiInfo, len(m))
	return m
}

// 把最终的映射写回 data/oss-images.json
func saveMapping(dataDir string, m ImageMap) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	out := filepath.Join(dataDir, "oss-images.json")
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(b, '\n'), 0o644)
}

// 构造 OSS 客户端，使用环境变量凭证
func newOSSClient(region string) (*oss.Client, error) {
	cred := credentials.NewEnvironmentVariableCredentialsProvider()
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(cred).
		WithRegion(region)
	return oss.NewClient(cfg), nil
}

// 把单个图片上传到 OSS；上传成功返回 (ossKey, url, true)。
func uploadOne(client *oss.Client, bucket, prefix, customDomain, region, abs string) (string, string, error) {
	hash, err := shortHash(abs)
	if err != nil {
		return "", "", err
	}
	base := filepath.Base(abs)
	ossKey := fmt.Sprintf("%s%s_%s", prefix, hash, base)
	url := buildURL(ossKey, customDomain, bucket, region)

	fmt.Printf("%s 上传 %s → oss://%s/%s\n", emojiUp, abs, bucket, ossKey)

	body, err := os.Open(abs)
	if err != nil {
		return "", "", err
	}
	defer body.Close()

	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(ossKey),
		Body:   body,
	}
	if _, err := client.PutObject(context.Background(), req); err != nil {
		return "", "", err
	}
	fmt.Printf("%s 上传完成 %s\n", emojiOK, url)
	return ossKey, url, nil
}

func main() {
	fmt.Println("🚀 OSS 图片上传工具启动")

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("%s 配置错误：%v\n", emojiError, err)
		os.Exit(1)
	}
	fmt.Printf("%s 配置：region=%s bucket=%s prefix=%s content=%s data=%s\n",
		emojiInfo, cfg.Region, cfg.Bucket, cfg.Prefix, cfg.ContentDir, cfg.DataDir)

	client, err := newOSSClient(cfg.Region)
	if err != nil {
		fmt.Printf("%s 初始化 OSS 客户端失败：%v\n", emojiError, err)
		os.Exit(1)
	}

	// 1. 收集所有 .md 文件里出现的本地图片
	mdFiles, err := findMarkdownFiles(cfg.ContentDir)
	if err != nil {
		fmt.Printf("%s 扫描 content 目录失败：%v\n", emojiError, err)
		os.Exit(1)
	}
	fmt.Printf("%s 发现 %d 个 Markdown 文件\n", emojiInfo, len(mdFiles))

	images := map[string]struct{}{}
	for _, md := range mdFiles {
		b, err := os.ReadFile(md)
		if err != nil {
			fmt.Printf("%s 读取 %s 失败：%v\n", emojiWarn, md, err)
			continue
		}
		for _, img := range extractLocalImages(string(b), md, cfg) {
			images[img] = struct{}{}
		}
	}
	fmt.Printf("%s 共发现 %d 个唯一本地图片\n", emojiInfo, len(images))

	if len(images) == 0 {
		fmt.Printf("%s 没有可上传的图片，结束\n", emojiInfo)
		return
	}

	// 2. 读取历史映射，复用已有的 oss_key/url
	mapping := loadExisting(cfg.DataDir)

	// 3. 遍历图片：跳过已存在且 key 一致的；其余上传并写入映射
	for img := range images {
		// 跳过空路径（防御性，正常不会出现）
		if strings.TrimSpace(img) == "" {
			continue
		}

		hash, err := shortHash(img)
		if err != nil {
			fmt.Printf("%s 计算 %s 哈希失败：%v\n", emojiError, img, err)
			continue
		}
		expectedKey := fmt.Sprintf("%s%s_%s", cfg.Prefix, hash, filepath.Base(img))

		if old, ok := mapping[img]; ok && old.OSSKey == expectedKey {
			fmt.Printf("%s 已存在映射，跳过上传 %s\n", emojiSkip, img)
			continue
		}

		ossKey, url, err := uploadOne(client, cfg.Bucket, cfg.Prefix, cfg.CustomDomain, cfg.Region, img)
		if err != nil {
			fmt.Printf("%s 上传 %s 失败：%v\n", emojiError, img, err)
			continue
		}
		mapping[img] = ImageEntry{OSSKey: ossKey, URL: url}
	}

	// 4. 写回映射
	if err := saveMapping(cfg.DataDir, mapping); err != nil {
		fmt.Printf("%s 写入映射文件失败：%v\n", emojiError, err)
		os.Exit(1)
	}
	fmt.Printf("%s 映射已写入 %s/oss-images.json，共 %d 条\n",
		emojiOK, cfg.DataDir, len(mapping))
}

// 递归扫描目录，返回所有 .md 文件的绝对路径
func findMarkdownFiles(root string) ([]string, error) {
	var out []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".md") {
			abs, err := filepath.Abs(p)
			if err != nil {
				return err
			}
			out = append(out, abs)
		}
		return nil
	})
	return out, err
}