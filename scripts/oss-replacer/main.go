// Package main 实现「CI 专用」的 OSS 链接替换工具。
//
// 背景：static/ 目录未纳入 git，GitHub Actions 检出代码后根本没有图片文件，
// 本地那套「render hook + data/oss-images.json」的方案在 CI 里必然失效
//（映射文件被 gitignore，图片也不在仓库里）。
//
// 本工具换个思路：列举远程 OSS 上已有的图片对象，按文件名反查出公开 URL，
// 然后把工作区里 Markdown 中的本地图片链接替换成 OSS URL，再交给 Hugo 构建。
//
// 两条硬约束：
//  1. 只按文件名匹配。CI 里没有本地图片文件，算不出 MD5，无法像上传工具那样
//     用内容哈希比对，只能拿 md 里的文件名去 OSS 对象列表里反查。
//     OSS key 形如 blog-images/{md5前12位}_{原文件名}。
//  2. 只改工作区里的临时副本。CI 每次构建都是重新检出，替换结果不会回流到仓库；
//     但在本地跑本工具会**真实改写 md 源文件**，跑完记得 git checkout -- content/ 还原。
//
// 与 scripts/oss-uploader 是两个独立程序、互不 import（正则各持一份副本）：
// 上传是写操作、替换是读操作，分开演进、分开授权更清晰。
package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 工具日志 emoji 前缀（与 oss-uploader 保持一致）
const (
	emojiOK    = "✅"
	emojiInfo  = "ℹ️ "
	emojiWarn  = "⚠️ "
	emojiError = "❌"
	emojiSwap  = "🔗"
)

// 每次列举的对象数量上限。
// 注意 ListObjectsV2Request.MaxKeys 是 int32 而非指针，不要写成 oss.Ptr(1000)。
// 不显式设置的话 SDK 会置 0，退化成服务端默认的每页 100 条。
const listPageSize = 1000

// 与 scripts/oss-uploader/main.go 保持同步的三个正则。
// 修改其中任一处时，另一处也要跟着改，否则「上传时识别的引用」和
// 「替换时改写的引用」会对不上。
var (
	// 匹配 Markdown 图片语法：![alt](path "title")
	//  - alt 可为空
	//  - path 不可包含空白或右括号
	//  - title 可选，由英文双引号包裹
	imageRegex = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)`)

	// 匹配指向本地图片的 Markdown 链接：[text](path)，用于渲染时生成 <a href>
	// 仅匹配目标为图片扩展名的链接，避免误处理站内导航链接
	imageLinkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+\.(?:png|jpg|jpeg|gif|svg|webp|bmp|ico))(?:\s+"([^"]*)")?\)`)

	// 匹配 Hugo figure / img shortcode 中的 src 属性
	//   {{< figure src="/ox-hugo/foo.png" >}}
	//   {{< img src="images/foo.png" >}}
	// 支持双引号或单引号包裹的 src 值
	figureSrcRegex = regexp.MustCompile(`\{\{<\s*(?:figure|img)\s+[^}>]*?src=["']([^"']+)["']`)
)

// 支持的图片扩展名（小写），与 oss-uploader 的白名单一致
var supportedExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".svg": true, ".webp": true,
	".bmp": true, ".ico": true,
}

// OSS key 的 basename 形如 {12位md5}_{原文件名}。
// 用这个正则把 md5 前缀剥掉还原出原文件名。
// 不能用「按第一个下划线切分」的朴素做法：原文件名本身常含下划线
//（例如 2026-06-17_14-38-39_screenshot.png），而 md5 部分绝不含下划线，
// 所以按正则锚定 12 位十六进制 + 下划线更稳。
var hashPrefixRegex = regexp.MustCompile(`^[0-9a-f]{12}_(.+)$`)

// 从环境变量读取并组装配置
type Config struct {
	Region       string // OSS 地域
	Bucket       string // OSS bucket
	Prefix       string // OSS 存储前缀
	CustomDomain string // CDN 自定义域名（可选）
	ContentDir   string // Hugo content 根目录
	Strict       bool   // 图片缺失时是否以失败退出
}

// 配置默认值
func defaultConfig() Config {
	return Config{
		Region:     "cn-hangzhou",
		Prefix:     "blog-images/",
		ContentDir: "./content",
	}
}

// 加载环境变量，未填的走默认；必填字段缺失则报错退出
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
	// STRICT 接受 1/true/yes（大小写不敏感），其余值一律视为关闭
	switch strings.ToLower(strings.TrimSpace(os.Getenv("STRICT"))) {
	case "1", "true", "yes":
		c.Strict = true
	}
	if c.Bucket == "" {
		return c, fmt.Errorf("OSS_BUCKET_NAME 环境变量未设置")
	}
	if !strings.HasSuffix(c.Prefix, "/") {
		c.Prefix += "/"
	}
	return c, nil
}

// 一个 OSS 对象：key、可公开访问的 URL、最后修改时间（用于同名冲突时择一）
type ossObject struct {
	key          string
	url          string
	lastModified time.Time
}

// 判断路径是否为远程 URL（http/https/data:），是则跳过替换。
//
// 这是幂等性的关键：三个正则是依次作用在同一段文本上的，
// imageRegex 把 ![](a.png) 换成 ![](https://.../a.png) 之后，
// 后面的 imageLinkRegex 依然能匹配到它（结尾仍是 .png）。
// 每个回调都先过这道判断，重复运行才不会产生二次替换。
func isRemoteURL(p string) bool {
	lp := strings.ToLower(p)
	return strings.HasPrefix(lp, "http://") ||
		strings.HasPrefix(lp, "https://") ||
		strings.HasPrefix(lp, "data:")
}

// 构造 OSS 客户端，使用环境变量凭证（只读列举即可，不需要写权限）
func newOSSClient(region string) *oss.Client {
	cred := credentials.NewEnvironmentVariableCredentialsProvider()
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(cred).
		WithRegion(region)
	return oss.NewClient(cfg)
}

// 根据 OSS key 与可选的自定义 CDN 域名，组装最终访问 URL。
// 规则与 scripts/oss-uploader 的 buildURL 完全一致，保证同一张图
// 本地渲染和 CI 替换拿到的是同一个地址。
func buildURL(ossKey, customDomain, bucket, region string) string {
	if customDomain != "" {
		base := strings.TrimRight(customDomain, "/")
		return base + "/" + ossKey
	}
	return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", bucket, region, ossKey)
}

// 从 OSS key 还原出原始文件名。
// 例：blog-images/0b294da7122b_foo.png -> foo.png
// key 不含 md5 前缀时（例如手动上传的对象）回退为整个 basename。
func originalName(ossKey string) string {
	base := path.Base(ossKey)
	if m := hashPrefixRegex.FindStringSubmatch(base); m != nil {
		return m[1]
	}
	return base
}

// 分页列举 OSS 上 prefix 下的全部对象，按「原始文件名」建索引。
//
// 同名不同内容的图片会对应多个 key（md5 前缀不同），全部保留在切片里，
// 由 pickObject 在使用时按确定性规则挑选并告警。
func listObjects(ctx context.Context, client *oss.Client, cfg Config) (map[string][]ossObject, error) {
	index := make(map[string][]ossObject)
	paginator := client.NewListObjectsV2Paginator(&oss.ListObjectsV2Request{
		Bucket:  oss.Ptr(cfg.Bucket),
		Prefix:  oss.Ptr(cfg.Prefix),
		MaxKeys: listPageSize,
	})
	for paginator.HasNext() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("列举 OSS 对象失败（Bucket=%s Prefix=%s）：%w", cfg.Bucket, cfg.Prefix, err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			// SDK 内部强制 encoding-type=url 并已自动解码，这里拿到的就是原始 key
			key := *obj.Key
			entry := ossObject{
				key: key,
				url: buildURL(key, cfg.CustomDomain, cfg.Bucket, cfg.Region),
			}
			if obj.LastModified != nil {
				entry.lastModified = *obj.LastModified
			}
			name := originalName(key)
			index[name] = append(index[name], entry)
		}
	}
	return index, nil
}

// 同名冲突时挑一个对象：最后修改时间新的优先，时间相同则按 key 升序。
// 两条规则叠加后在给定 bucket 状态下结果完全确定，不会因 map 遍历顺序抖动。
// 理由：同一个文件名被反复上传时，最新的那份通常才是作者想要的。
func pickObject(cands []ossObject) ossObject {
	if len(cands) == 1 {
		return cands[0]
	}
	sorted := make([]ossObject, len(cands))
	copy(sorted, cands)
	sort.Slice(sorted, func(i, j int) bool {
		ti, tj := sorted[i].lastModified, sorted[j].lastModified
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return sorted[i].key < sorted[j].key
	})
	return sorted[0]
}

// 把整段正则匹配中的「路径捕获组」替换为 newPath，其余部分原样保留。
//
// 不能简单地对整段匹配做 strings.Replace：像 ![foo.png](foo.png) 这种
// alt 与 path 同名的情况，替换会错落到 alt 上。这里用捕获组的下标精确定位。
func replacePathInMatch(re *regexp.Regexp, match string, group int, newPath string) string {
	loc := re.FindStringSubmatchIndex(match)
	if loc == nil || len(loc) <= group*2+1 {
		return match
	}
	start, end := loc[group*2], loc[group*2+1]
	if start < 0 || end < 0 {
		return match
	}
	return match[:start] + newPath + match[end:]
}

// 从 md 里的引用路径取出用于反查的文件名。
// md 里可能是 URL 编码形式（ox-hugo 导出空格、中文时会出现 %20 之类），
// 先尝试解码；解码失败或解码后为空则回退原始串。
func lookupName(ref string) string {
	decoded := ref
	if unescaped, err := url.PathUnescape(ref); err == nil && unescaped != "" {
		decoded = unescaped
	}
	return path.Base(decoded)
}

// 单个文件的替换结果
type fileResult struct {
	replaced int            // 成功替换的引用数
	missing  map[string]int // 未在 OSS 上找到的文件名 -> 出现次数
}

// 对一段 Markdown 文本依次应用三个正则，把命中的本地图片引用换成 OSS URL。
// 替换过程中缺失的文件名累加进 res.missing，同名冲突只在首次出现时告警
// （warned 用来去重，避免同一文件名在每个文件里都刷一遍日志）。
func applyReplacements(text string, index map[string][]ossObject, res *fileResult, warned map[string]bool) string {
	// 生成一个替换回调。group 是「路径」在该正则里的捕获组序号。
	replacer := func(re *regexp.Regexp, group int) func(string) string {
		return func(match string) string {
			sub := re.FindStringSubmatch(match)
			if sub == nil || len(sub) <= group {
				return match
			}
			ref := sub[group]

			// 幂等守卫：已经是远程地址就原样返回，绝不二次替换
			if isRemoteURL(ref) {
				return match
			}
			// 与上传工具一致：只认白名单里的图片扩展名
			if !supportedExts[strings.ToLower(path.Ext(ref))] {
				return match
			}

			name := lookupName(ref)
			cands, ok := index[name]
			if !ok || len(cands) == 0 {
				res.missing[name]++
				return match
			}

			if len(cands) > 1 && !warned[name] {
				warned[name] = true
				fmt.Printf("%s 文件名冲突 %q 命中 %d 个 OSS 对象，选用最新的一个：\n", emojiWarn, name, len(cands))
				for _, c := range cands {
					fmt.Printf("      %s\n", c.url)
				}
			}

			chosen := pickObject(cands)
			res.replaced++
			return replacePathInMatch(re, match, group, chosen.url)
		}
	}

	// 顺序与 oss-uploader 的扫描顺序一致：图片语法 → 图片链接 → figure shortcode。
	// 每个回调开头的 isRemoteURL 守卫保证前一步的结果不会被后一步重复改写。
	text = imageRegex.ReplaceAllStringFunc(text, replacer(imageRegex, 2))
	text = imageLinkRegex.ReplaceAllStringFunc(text, replacer(imageLinkRegex, 2))
	text = figureSrcRegex.ReplaceAllStringFunc(text, replacer(figureSrcRegex, 1))
	return text
}

// 改写单个 md 文件：读盘 → 替换 → 仅在内容真的变了时把新内容交给调用方写回。
// 返回 (替换结果, 新内容, 是否有改动, 错误)。
func replaceInFile(mdPath string, index map[string][]ossObject, warned map[string]bool) (fileResult, []byte, bool, error) {
	raw, err := os.ReadFile(mdPath)
	if err != nil {
		return fileResult{}, nil, false, err
	}

	res := fileResult{missing: map[string]int{}}
	out := []byte(applyReplacements(string(raw), index, &res, warned))

	if bytes.Equal(raw, out) {
		return res, nil, false, nil
	}
	return res, out, true, nil
}

// 递归扫描目录，返回所有 .md 文件的路径
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
			out = append(out, p)
		}
		return nil
	})
	return out, err
}

// 写回文件，沿用原文件权限，避免把可执行位或只读位改掉
func writeFilePreserveMode(mdPath string, content []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(mdPath); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(mdPath, content, mode)
}

func main() {
	fmt.Println("🚀 OSS 链接替换工具启动")

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("%s 配置错误：%v\n", emojiError, err)
		os.Exit(1)
	}
	fmt.Printf("%s 配置：region=%s bucket=%s prefix=%s content=%s strict=%v\n",
		emojiInfo, cfg.Region, cfg.Bucket, cfg.Prefix, cfg.ContentDir, cfg.Strict)

	client := newOSSClient(cfg.Region)

	// 1. 列举 OSS 上已有的图片，按原始文件名建索引
	index, err := listObjects(context.Background(), client, cfg)
	if err != nil {
		fmt.Printf("%s %v\n", emojiError, err)
		// 403 是本步骤最常见的失败原因，且 OSS 返回的原文极具误导性，
		// 这里补上两条最可能的排查方向，省得对着 "bucket does not belong to you" 发呆。
		fmt.Printf("%s 列举失败通常是这两个原因之一：\n", emojiInfo)
		fmt.Printf("     1. OSS_REGION 与 bucket 实际所在地域不一致（报错含 'must be addressed using the specified endpoint'）；\n")
		fmt.Printf("     2. RAM 子账号没有 oss:ListObjects 权限（报错含 'AccessDenied' / 'does not belong to you'）。\n")
		fmt.Printf("        注意：本地上传只需要 oss:PutObject，本工具额外要求 oss:ListObjects。\n")
		os.Exit(1)
	}
	fmt.Printf("%s 已在 OSS 上找到 %d 个对象（按文件名去重后 %d 个名字）\n",
		emojiInfo, countObjects(index), len(index))
	if len(index) == 0 {
		// 一个都没有通常不是「文章里没图」，而是前缀写错、bucket 写错，
		// 或者 RAM 子账号没有 oss:ListObjects 权限。给足排查线索。
		fmt.Printf("%s 未列举到任何对象，请检查 OSS_BUCKET_NAME/OSS_IMAGE_PREFIX 是否正确，以及凭证是否具备 oss:ListObjects 权限\n", emojiWarn)
		if cfg.Strict {
			os.Exit(1)
		}
	}

	// 2. 遍历 md 文件做替换
	mdFiles, err := findMarkdownFiles(cfg.ContentDir)
	if err != nil {
		fmt.Printf("%s 扫描 content 目录失败：%v\n", emojiError, err)
		os.Exit(1)
	}
	fmt.Printf("%s 发现 %d 个 Markdown 文件\n", emojiInfo, len(mdFiles))

	warned := map[string]bool{}
	totalReplaced, totalChanged := 0, 0
	missing := map[string]int{}

	for _, md := range mdFiles {
		res, content, changed, err := replaceInFile(md, index, warned)
		if err != nil {
			fmt.Printf("%s 读取 %s 失败：%v\n", emojiWarn, md, err)
			continue
		}
		if !changed {
			continue
		}
		if err := writeFilePreserveMode(md, content); err != nil {
			fmt.Printf("%s 写入 %s 失败：%v\n", emojiError, md, err)
			os.Exit(1)
		}
		totalChanged++
		totalReplaced += res.replaced
		fmt.Printf("%s %s：替换 %d 处\n", emojiSwap, md, res.replaced)
		for name, n := range res.missing {
			missing[name] += n
		}
	}

	// 3. 汇总
	fmt.Printf("%s 共改写 %d 个文件、替换 %d 处图片链接\n", emojiOK, totalChanged, totalReplaced)

	if len(missing) > 0 {
		fmt.Printf("%s 以下 %d 个图片在 OSS 上找不到对应对象（多半是忘了先在本地跑 make build 上传）：\n",
			emojiWarn, len(missing))
		for name, n := range missing {
			fmt.Printf("      %s（出现 %d 次）\n", name, n)
		}
		fmt.Printf("   这些引用保持本地路径不变，站点上会表现为坏图。\n")
		if cfg.Strict {
			fmt.Printf("%s STRICT=1，按失败处理\n", emojiError)
			os.Exit(1)
		}
	}
}

// 统计索引里的对象总数（同名冲突会让总数大于名字数）
func countObjects(index map[string][]ossObject) int {
	n := 0
	for _, cands := range index {
		n += len(cands)
	}
	return n
}
