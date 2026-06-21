# 小说→分镜（前半段）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 上传 .txt 小说 → 正则切章 → 每章 LLM 抽 9 个可编辑分镜（场景/人物/提示词/对白/镜头），存 MySQL，前端可查看编辑。

**Architecture:** 后端新增 `internal/novel/`（切章+LLM 抽取纯逻辑，可单测）+ `internal/api/novel_handlers.go`（HTTP），复用现有 GORM/MySQL/credits/异步任务/OSS-less（原文存 DB）。LLM 走中转站 chat（`OPENAI_BASE_URL`）。前端新增 3 个页面 + Pinia store + composable。编排全在后端，前端触发+轮询+编辑。

**Tech Stack:** Go (Gin/GORM/MySQL)、Vue 3 (Pinia/Naive UI/Vue Router)、中转站 OpenAI 兼容 chat API。

**Spec:** [docs/superpowers/specs/2026-06-21-novel-storyboard-design.md](../specs/2026-06-21-novel-storyboard-design.md)

**测试策略:** 纯逻辑（切章正则、分镜 JSON 解析）用 `go test` 单测；handler/前端用运行验证（后端 :8092 + 前端 :5173 已在跑，curl + 浏览器）。

---

## 文件结构

**后端（新建/修改）**
- 新建 `backend/migrations/027_add_novel_system.sql` — 3 张表 DDL
- 新建 `backend/internal/db/novel_models.go` — GORM 模型 Novel/NovelChapter/NovelShot
- 修改 `backend/internal/config/config.go` — 3 个 getter
- 新建 `backend/internal/novel/chapter.go` + `chapter_test.go` — 正则切章（纯逻辑+单测）
- 新建 `backend/internal/novel/chat.go` — relayChat helper
- 新建 `backend/internal/novel/storyboard.go` + `storyboard_test.go` — LLM 抽分镜 + JSON 解析单测
- 新建 `backend/internal/api/novel_handlers.go` — 7 个 handler
- 修改 `backend/main.go` — 注册 `/api/novel` 路由

**前端（新建/修改）**
- 新建 `frontend/src/stores/novel.js`
- 新建 `frontend/src/composables/useNovel.js`
- 新建 `frontend/src/views/NovelList.vue` / `NovelDetail.vue` / `ChapterStoryboard.vue`
- 修改 `frontend/src/router/index.js` — 3 条路由
- 修改 `frontend/src/components/AppSidebar.vue` — 导航项
- 修改 `frontend/src/locales/zh.json` / `en.json` — i18n

---

## Task 1: 数据库迁移 + GORM 模型

**Files:**
- Create: `backend/migrations/027_add_novel_system.sql`
- Create: `backend/internal/db/novel_models.go`

- [ ] **Step 1: 写迁移 SQL**

`backend/migrations/027_add_novel_system.sql`:
```sql
-- 小说系统：小说 / 章节 / 分镜
CREATE TABLE IF NOT EXISTS `novels` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint unsigned NOT NULL,
  `title` varchar(200) NOT NULL,
  `source_filename` varchar(255) DEFAULT NULL,
  `raw_content` longtext,
  `chapter_count` int NOT NULL DEFAULT 0,
  `status` varchar(20) NOT NULL DEFAULT 'active',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_novels_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说';

CREATE TABLE IF NOT EXISTS `novel_chapters` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `novel_id` bigint unsigned NOT NULL,
  `chapter_index` int NOT NULL,
  `title` varchar(200) DEFAULT NULL,
  `content` longtext,
  `storyboard_status` varchar(20) NOT NULL DEFAULT 'none',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_novel_chapter` (`novel_id`,`chapter_index`),
  KEY `idx_novel_chapters_novel_id` (`novel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说章节';

CREATE TABLE IF NOT EXISTS `novel_shots` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `chapter_id` bigint unsigned NOT NULL,
  `shot_index` int NOT NULL,
  `scene` varchar(1000) DEFAULT NULL,
  `characters` varchar(500) DEFAULT NULL,
  `prompt` varchar(2000) DEFAULT NULL,
  `dialogue` varchar(1000) DEFAULT NULL,
  `camera` varchar(100) DEFAULT NULL,
  `image_url` varchar(500) DEFAULT NULL,
  `video_url` varchar(500) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_chapter_shot` (`chapter_id`,`shot_index`),
  KEY `idx_novel_shots_chapter_id` (`chapter_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='小说分镜';
```

- [ ] **Step 2: 写 GORM 模型**

`backend/internal/db/novel_models.go`:
```go
package db

import "time"

type Novel struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UserID        uint64    `gorm:"type:bigint;index;not null" json:"user_id"`
	Title         string    `gorm:"type:varchar(200);not null" json:"title"`
	SourceFilename string   `gorm:"type:varchar(255)" json:"source_filename"`
	RawContent    string    `gorm:"type:longtext" json:"-"`
	ChapterCount  int       `gorm:"type:int;not null;default:0" json:"chapter_count"`
	Status        string    `gorm:"type:varchar(20);default:'active'" json:"status"`
	CreatedAt     time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"type:datetime" json:"updated_at"`
}

type NovelChapter struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	NovelID          uint64    `gorm:"type:bigint;index;not null" json:"novel_id"`
	ChapterIndex     int       `gorm:"type:int;not null" json:"chapter_index"`
	Title            string    `gorm:"type:varchar(200)" json:"title"`
	Content          string    `gorm:"type:longtext" json:"content"`
	StoryboardStatus string    `gorm:"type:varchar(20);default:'none'" json:"storyboard_status"`
	CreatedAt        time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"type:datetime" json:"updated_at"`
}

type NovelShot struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	ChapterID  uint64    `gorm:"type:bigint;index;not null" json:"chapter_id"`
	ShotIndex  int       `gorm:"type:int;not null" json:"shot_index"`
	Scene      string    `gorm:"type:varchar(1000)" json:"scene"`
	Characters string    `gorm:"type:varchar(500)" json:"characters"`
	Prompt     string    `gorm:"type:varchar(2000)" json:"prompt"`
	Dialogue   string    `gorm:"type:varchar(1000)" json:"dialogue"`
	Camera     string    `gorm:"type:varchar(100)" json:"camera"`
	ImageURL   string    `gorm:"type:varchar(500)" json:"image_url"`
	VideoURL   string    `gorm:"type:varchar(500)" json:"video_url"`
	CreatedAt  time.Time `gorm:"type:datetime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"type:datetime" json:"updated_at"`
}
```

- [ ] **Step 3: 重启后端验证迁移 applied**

Run: `go build -C d:\code\xiaoye-ai\backend ./...`（编译通过）
然后重启后端，查日志出现 `migrations: applied 027_add_novel_system.sql`。
MySQL 里确认：`SHOW TABLES LIKE 'novel%';` → novels / novel_chapters / novel_shots。

- [ ] **Step 4: Commit**

```bash
git add backend/migrations/027_add_novel_system.sql backend/internal/db/novel_models.go
git commit -m "feat(novel): add novel/chapter/shot tables and models"
```

---

## Task 2: 配置 getter

**Files:**
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: 追加 3 个 getter**

在 `config.go` 末尾追加：
```go
// GetNovelLLMModel 抽分镜用的中转站 chat 模型名。
func GetNovelLLMModel() string {
	m := os.Getenv("NOVEL_LLM_MODEL")
	if m == "" {
		m = "deepseek-v4-flash"
	}
	return m
}

// GetNovelStoryboardCredits 每章抽分镜扣钻数。
func GetNovelStoryboardCredits() int {
	v := os.Getenv("NOVEL_STORYBOARD_CREDITS")
	if v == "" {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 1
	}
	return n
}

// GetNovelChapterMaxChars 送 LLM 的章节正文上限。
func GetNovelChapterMaxChars() int {
	v := os.Getenv("NOVEL_CHAPTER_MAX_CHARS")
	if v == "" {
		return 12000
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 12000
	}
	return n
}
```

- [ ] **Step 2: 同步 .env / .env.example**

在 `backend/.env` 和 `backend/.env.example` 追加：
```
# ---- 小说分镜 ----
NOVEL_LLM_MODEL=deepseek-v4-flash
NOVEL_STORYBOARD_CREDITS=1
NOVEL_CHAPTER_MAX_CHARS=12000
```

- [ ] **Step 3: 编译验证**

Run: `go build -C d:\code\xiaoye-ai\backend ./...` → 通过。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/config/config.go backend/.env.example
git commit -m "feat(novel): add config getters for LLM model/credits/maxchars"
```
（.env 不提交，gitignored）

---

## Task 3: 正则切章（纯逻辑 + 单测）

**Files:**
- Create: `backend/internal/novel/chapter.go`
- Create: `backend/internal/novel/chapter_test.go`

- [ ] **Step 1: 先写失败的单测**

`backend/internal/novel/chapter_test.go`:
```go
package novel

import "testing"

func TestParseChapters_MultiChapter(t *testing.T) {
	raw := "第一章 初遇\n小明遇见了小红。\n第二章 重逢\n十年后他们再次相遇。"
	cs := ParseChapters(raw)
	if len(cs) != 2 {
		t.Fatalf("want 2 chapters, got %d", len(cs))
	}
	if cs[0].Title != "第一章 初遇" {
		t.Errorf("title0=%q", cs[0].Title)
	}
	if cs[0].Content != "小明遇见了小红。" {
		t.Errorf("content0=%q", cs[0].Content)
	}
	if cs[1].Index != 2 || cs[1].Title != "第二章 重逢" {
		t.Errorf("chapter2 wrong: %+v", cs[1])
	}
}

func TestParseChapters_NoMarker_WholeAsOne(t *testing.T) {
	raw := "这是一段没有章节标题的文字。继续。"
	cs := ParseChapters(raw)
	if len(cs) != 1 {
		t.Fatalf("want 1 chapter, got %d", len(cs))
	}
	if cs[0].Title != "全文" {
		t.Errorf("title=%q want 全文", cs[0].Title)
	}
	if cs[0].Content != raw {
		t.Errorf("content should be whole text")
	}
}

func TestParseChapters_ArabicNumerals(t *testing.T) {
	raw := "第1章 开端\n内容A\n第2章 发展\n内容B"
	cs := ParseChapters(raw)
	if len(cs) != 2 || cs[0].Title != "第1章 开端" || cs[1].Title != "第2章 发展" {
		t.Errorf("arabic numeral parse wrong: %+v", cs)
	}
}

func TestParseChapters_Hui(t *testing.T) {
	raw := "第一回 风雪夜\n内容。\n第二回 归途\n内容2。"
	cs := ParseChapters(raw)
	if len(cs) != 2 {
		t.Fatalf("want 2 (回), got %d", len(cs))
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -C d:\code\xiaoye-ai\backend ./internal/novel/ -run TestParseChapters -v`
Expected: FAIL（`ParseChapters` 未定义 / 包不存在）。

- [ ] **Step 3: 实现 chapter.go**

`backend/internal/novel/chapter.go`:
```go
package novel

import (
	"regexp"
	"strings"
)

// chapterTitleRe 匹配行首的章节标题：第一章 / 第123章 / 第二节 / 第一回 ...
var chapterTitleRe = regexp.MustCompile(`(?m)^[[:space:]]*第[0-9一二三四五六七八九十百千零〇两]+[章节回卷部篇][^.\n]*$`)

// ParsedChapter 切出来的章节。
type ParsedChapter struct {
	Index   int
	Title   string
	Content string
}

// ParseChapters 用正则把小说正文切成章节。未匹配到任何标题 → 整文作为 1 章（title=全文）。
func ParseChapters(raw string) []ParsedChapter {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	locs := chapterTitleRe.FindAllStringIndex(raw, -1)
	if len(locs) == 0 {
		return []ParsedChapter{{Index: 1, Title: "全文", Content: strings.TrimSpace(raw)}}
	}
	out := make([]ParsedChapter, 0, len(locs))
	for i, start := range locs {
		end := len(raw)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		seg := raw[start:end]
		title := seg
		body := ""
		if nl := strings.IndexByte(seg, '\n'); nl >= 0 {
			title = seg[:nl]
			body = seg[nl+1:]
		}
		out = append(out, ParsedChapter{
			Index:   i + 1,
			Title:   strings.TrimSpace(title),
			Content: strings.TrimSpace(body),
		})
	}
	return out
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -C d:\code\xiaoye-ai\backend ./internal/novel/ -run TestParseChapters -v`
Expected: PASS（4 个用例全过）。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/novel/chapter.go backend/internal/novel/chapter_test.go
git commit -m "feat(novel): regex chapter parser with tests"
```

---

## Task 4: 中转站 chat helper

**Files:**
- Create: `backend/internal/novel/chat.go`

- [ ] **Step 1: 实现 relayChat**

`backend/internal/novel/chat.go`:
```go
package novel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type relayChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RelayChat 调中转站 chat/completions（OPENAI_BASE_URL），返回 assistant 文本。
// response_format=json_object，要求模型返回 JSON 文本。
func RelayChat(model, systemPrompt, userContent string) (string, error) {
	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if baseURL == "" || apiKey == "" {
		return "", fmt.Errorf("中转站未配置 (OPENAI_BASE_URL / OPENAI_API_KEY)")
	}

	body := map[string]interface{}{
		"model": model,
		"messages": []relayChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		"response_format": map[string]string{"type": "json_object"},
		"stream":          false,
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用失败: %v", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("中转站返回 %d: %s", resp.StatusCode, truncate(string(rb), 300))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return "", fmt.Errorf("响应解析失败: %v", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("中转站错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("中转站未返回内容")
	}
	return parsed.Choices[0].Message.Content, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
```

- [ ] **Step 2: 编译验证**

Run: `go build -C d:\code\xiaoye-ai\backend ./...` → 通过。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/novel/chat.go
git commit -m "feat(novel): relay chat helper for LLM calls"
```

---

## Task 5: 分镜抽取 + JSON 解析（单测）

**Files:**
- Create: `backend/internal/novel/storyboard.go`
- Create: `backend/internal/novel/storyboard_test.go`

- [ ] **Step 1: 先写 JSON 解析单测（纯函数，不联网）**

`backend/internal/novel/storyboard_test.go`:
```go
package novel

import "testing"

func TestParseShotsJSON_Valid(t *testing.T) {
	raw := `{"shots":[{"scene":"教室","characters":"小明,老师","prompt":"教室阳光","dialogue":"你好","camera":"推进"},{"scene":"操场","characters":"小明","prompt":"操场奔跑","dialogue":"","camera":"平移"}]}`
	shots, err := parseShotsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(shots) != 2 {
		t.Fatalf("want 2, got %d", len(shots))
	}
	if shots[0].Scene != "教室" || shots[0].Prompt != "教室阳光" {
		t.Errorf("shot0 wrong: %+v", shots[0])
	}
}

func TestParseShotsJSON_TruncateTo9(t *testing.T) {
	// 构造 12 个 shot，应截断为 9
	raw := `{"shots":[`
	for i := 0; i < 12; i++ {
		if i > 0 {
			raw += ","
		}
		raw += `{"scene":"s` + itoa(i) + `","prompt":"p"}`
	}
	raw += `}`
	shots, err := parseShotsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(shots) != 9 {
		t.Errorf("want 9 (truncated), got %d", len(shots))
	}
}

func TestParseShotsJSON_CodeFence(t *testing.T) {
	raw := "```json\n{\"shots\":[{\"scene\":\"x\",\"prompt\":\"y\"}]}\n```"
	shots, err := parseShotsJSON(raw)
	if err != nil || len(shots) != 1 {
		t.Errorf("code-fence parse failed: %v, %d", err, len(shots))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test -C d:\code\xiaoye-ai\backend ./internal/novel/ -run TestParseShotsJSON -v`
Expected: FAIL（`parseShotsJSON` 未定义）。

- [ ] **Step 3: 实现 storyboard.go**

`backend/internal/novel/storyboard.go`:
```go
package novel

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"google-ai-proxy/internal/config"
)

// Shot 是从章节抽出的单个分镜（与 db.NovelShot 字段对齐，但不带 ID/时间戳）。
type Shot struct {
	Scene      string `json:"scene"`
	Characters string `json:"characters"`
	Prompt     string `json:"prompt"`
	Dialogue   string `json:"dialogue"`
	Camera     string `json:"camera"`
}

const storyboardSystemPrompt = `你是一名专业的影视分镜师。我会给你一章小说正文，请你把它拆成恰好 9 个分镜（镜头），覆盖本章主要情节。
严格返回 JSON，格式为：{"shots":[{"scene":"场景描述","characters":"出场人物，逗号分隔","prompt":"用于 AI 绘图的画面提示词（中文，描述这一镜的视觉画面，不含人物对白）","dialogue":"该镜头的重要对白或旁白，没有则空字符串","camera":"镜头运动，如 推进/平移/特写/全景/跟随"}, ...共9个...]}
只返回 JSON，不要任何解释。必须恰好 9 个。`

// ExtractShots 调 LLM 为一章正文抽取分镜。
func ExtractShots(chapterContent string) ([]Shot, error) {
	maxChars := config.GetNovelChapterMaxChars()
	content := chapterContent
	if len([]rune(content)) > maxChars {
		content = string([]rune(content)[:maxChars])
	}
	model := config.GetNovelLLMModel()
	raw, err := RelayChat(model, storyboardSystemPrompt, content)
	if err != nil {
		return nil, err
	}
	return parseShotsJSON(raw)
}

var jsonShotRe = regexp.MustCompile(`(?s)\{.*\}`)

// parseShotsJSON 从 LLM 返回文本里解析 shots（兼容裸 JSON / ```json 代码块）。
// 超过 9 个截断为 9；0 个报错。
func parseShotsJSON(raw string) ([]Shot, error) {
	raw = strings.TrimSpace(raw)
	// 剥 ```json ... ``` 包裹
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	// 兜底：取第一个 {...}
	if !strings.HasPrefix(raw, "{") {
		if m := jsonShotRe.FindString(raw); m != "" {
			raw = m
		}
	}
	var wrap struct {
		Shots []Shot `json:"shots"`
	}
	if err := json.Unmarshal([]byte(raw), &wrap); err != nil {
		return nil, fmt.Errorf("分镜 JSON 解析失败: %v (raw: %s)", err, truncate(raw, 200))
	}
	if len(wrap.Shots) == 0 {
		return nil, fmt.Errorf("LLM 未返回任何分镜")
	}
	if len(wrap.Shots) > 9 {
		wrap.Shots = wrap.Shots[:9]
	}
	return wrap.Shots, nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test -C d:\code\xiaoye-ai\backend ./internal/novel/ -v`
Expected: PASS（切章 4 个 + 解析 3 个全过）。

- [ ] **Step 5: Commit**

```bash
git add backend/internal/novel/storyboard.go backend/internal/novel/storyboard_test.go
git commit -m "feat(novel): storyboard extraction + JSON parse with tests"
```

---

## Task 6: Novel/Chapter CRUD handler

**Files:**
- Create: `backend/internal/api/novel_handlers.go`

- [ ] **Step 1: 实现 upload / list / detail / delete**

`backend/internal/api/novel_handlers.go`（先写 CRUD 部分，storyboard 在 Task 7）：
```go
package api

import (
	"io"
	"net/http"
	"strings"
	"time"

	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/novel"

	"github.com/gin-gonic/gin"
)

const novelMaxUploadBytes = 10 * 1024 * 1024 // 10MB

// UploadNovel POST /api/novel/upload  multipart file=.txt
func UploadNovel(c *gin.Context) {
	userID := c.GetUint64("userID")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 .txt 文件"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".txt") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .txt 文件"})
		return
	}
	if file.Size > novelMaxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大（上限 10MB）"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	content := string(raw)

	chapters := novel.ParseChapters(content)
	title := strings.TrimSuffix(file.Filename, ".txt")
	if title == "" {
		title = "未命名小说"
	}

	n := &db.Novel{
		UserID:         userID,
		Title:          title,
		SourceFilename: file.Filename,
		RawContent:     content,
		ChapterCount:   len(chapters),
		Status:         "active",
	}
	if err := db.DB.Create(n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	rows := make([]db.NovelChapter, 0, len(chapters))
	for _, ch := range chapters {
		rows = append(rows, db.NovelChapter{
			NovelID:          n.ID,
			ChapterIndex:     ch.Index,
			Title:            ch.Title,
			Content:          ch.Content,
			StoryboardStatus: "none",
		})
	}
	if len(rows) > 0 {
		if err := db.DB.Create(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存章节失败"})
			return
		}
	}

	resp := gin.H{"novel_id": n.ID, "title": n.Title, "chapter_count": n.ChapterCount}
	if len(chapters) == 1 && chapters[0].Title == "全文" {
		resp["warning"] = "未识别到章节标题，已按整文处理"
	}
	c.JSON(http.StatusOK, resp)
}

// ListNovels GET /api/novel
func ListNovels(c *gin.Context) {
	userID := c.GetUint64("userID")
	limit, offset := parseListPagination(c)
	var novels []db.Novel
	db.DB.Where("user_id = ? AND status = 'active'", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&novels)
	var total int64
	db.DB.Model(&db.Novel{}).Where("user_id = ? AND status = 'active'", userID).Count(&total)
	c.JSON(http.StatusOK, gin.H{"items": novels, "total": total, "limit": limit, "offset": offset})
}

// GetNovel GET /api/novel/:id
func GetNovel(c *gin.Context) {
	userID := c.GetUint64("userID")
	var n db.Novel
	if err := db.DB.First(&n, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "小说不存在"})
		return
	}
	if n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var chapters []db.NovelChapter
	db.DB.Where("novel_id = ?", n.ID).Order("chapter_index ASC").Find(&chapters)
	// 列表不返回 content（避免大字段）
	type chLite struct {
		ID               uint64 `json:"id"`
		ChapterIndex     int    `json:"chapter_index"`
		Title            string `json:"title"`
		StoryboardStatus string `json:"storyboard_status"`
	}
	lite := make([]chLite, 0, len(chapters))
	for _, ch := range chapters {
		lite = append(lite, chLite{ch.ID, ch.ChapterIndex, ch.Title, ch.StoryboardStatus})
	}
	c.JSON(http.StatusOK, gin.H{"novel": n, "chapters": lite})
}

// DeleteNovel DELETE /api/novel/:id
func DeleteNovel(c *gin.Context) {
	userID := c.GetUint64("userID")
	var n db.Novel
	if err := db.DB.First(&n, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "小说不存在"})
		return
	}
	if n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	// 级联删 chapters + shots
	var chIDs []uint64
	db.DB.Model(&db.NovelChapter{}).Where("novel_id = ?", n.ID).Pluck("id", &chIDs)
	if len(chIDs) > 0 {
		db.DB.Where("chapter_id IN ?", chIDs).Delete(&db.NovelShot{})
		db.DB.Where("novel_id = ?", n.ID).Delete(&db.NovelChapter{})
	}
	db.DB.Delete(&n)
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
```

> `parseListPagination` 已存在于 `inspiration_handlers.go`（按 limit/offset 解析），直接复用。

- [ ] **Step 2: 编译验证**

Run: `go build -C d:\code\xiaoye-ai\backend ./...` → 通过。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/api/novel_handlers.go
git commit -m "feat(novel): upload/list/detail/delete handlers"
```

---

## Task 7: 分镜 handler（抽取/查询/编辑）

**Files:**
- Modify: `backend/internal/api/novel_handlers.go`（追加 storyboard 部分）

- [ ] **Step 1: 追加 3 个 handler**

在 `novel_handlers.go` 末尾追加：
```go
// TriggerStoryboard POST /api/novel/chapter/:id/storyboard
func TriggerStoryboard(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))

	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	// 所有权：通过 novel 归属校验
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	if ch.StoryboardStatus == "extracting" {
		c.JSON(http.StatusConflict, gin.H{"error": "该章节正在抽取中"})
		return
	}

	credits := config.GetNovelStoryboardCredits()
	if credits > 0 {
		if _, ok := getActiveUser(c, userID); !ok {
			return
		}
		deduct := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, credits).
			Update("credits", gorm.Expr("credits - ?", credits))
		if deduct.Error != nil || deduct.RowsAffected == 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足"})
			return
		}
		if err := recordCreditTransaction(db.DB, userID, -credits, "novel_storyboard_cost", "novel", "", "分镜抽取"); err != nil {
			refundCredits(userID, credits, "novel-storyboard-ledger-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "记录流水失败"})
			return
		}
	}

	// 置 extracting，清旧分镜
	db.DB.Model(&ch).Update("storyboard_status", "extracting")
	db.DB.Where("chapter_id = ?", chapterID).Delete(&db.NovelShot{})

	go func(chapterID, userID uint64, credits int, content string) {
		var status string
		var errMsg string
		shots, err := novel.ExtractShots(content)
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			if credits > 0 {
				refundCredits(userID, credits, "novel-storyboard-failed")
			}
		} else {
			rows := make([]db.NovelShot, 0, len(shots))
			for i, s := range shots {
				rows = append(rows, db.NovelShot{
					ChapterID:  chapterID,
					ShotIndex:  i + 1,
					Scene:      s.Scene,
					Characters: s.Characters,
					Prompt:     s.Prompt,
					Dialogue:   s.Dialogue,
					Camera:     s.Camera,
				})
			}
			if err := db.DB.Create(&rows).Error; err != nil {
				status = "failed"
				errMsg = "保存分镜失败"
				if credits > 0 {
					refundCredits(userID, credits, "novel-storyboard-save-failed")
				}
			} else {
				status = "ready"
			}
		}
		upd := map[string]interface{}{"storyboard_status": status, "updated_at": time.Now()}
		_ = errMsg
		db.DB.Model(&db.NovelChapter{}).Where("id = ?", chapterID).Updates(upd)
	}(chapterID, userID, credits, ch.Content)

	c.JSON(http.StatusOK, gin.H{"status": "extracting"})
}

// GetStoryboard GET /api/novel/chapter/:id/storyboard
func GetStoryboard(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))
	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var shots []db.NovelShot
	db.DB.Where("chapter_id = ?", chapterID).Order("shot_index ASC").Find(&shots)
	c.JSON(http.StatusOK, gin.H{
		"status":      ch.StoryboardStatus,
		"chapter_id":  ch.ID,
		"title":       ch.Title,
		"content":     ch.Content,
		"shots":       shots,
	})
}

// UpdateShot PUT /api/novel/shot/:id
func UpdateShot(c *gin.Context) {
	userID := c.GetUint64("userID")
	shotID := parseUintParam(c.Param("id"))
	var shot db.NovelShot
	if err := db.DB.First(&shot, shotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分镜不存在"})
		return
	}
	// 所有权
	var ch db.NovelChapter
	if err := db.DB.First(&ch, shot.ChapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var req struct {
		Scene      *string `json:"scene"`
		Characters *string `json:"characters"`
		Prompt     *string `json:"prompt"`
		Dialogue   *string `json:"dialogue"`
		Camera     *string `json:"camera"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	updates := map[string]interface{}{}
	if req.Scene != nil {
		updates["scene"] = *req.Scene
	}
	if req.Characters != nil {
		updates["characters"] = *req.Characters
	}
	if req.Prompt != nil {
		updates["prompt"] = *req.Prompt
	}
	if req.Dialogue != nil {
		updates["dialogue"] = *req.Dialogue
	}
	if req.Camera != nil {
		updates["camera"] = *req.Camera
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无更新字段"})
		return
	}
	updates["updated_at"] = time.Now()
	db.DB.Model(&shot).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}
```

并在 `novel_handlers.go` import 块加入：
```go
"google-ai-proxy/internal/config"
"gorm.io/gorm"
```
并加 helper（若 api 包无 `parseUintParam`）：
```go
func parseUintParam(s string) uint64 {
	var n uint64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + uint64(ch-'0')
	}
	return n
}
```
（先 grep 确认 `parseUintParam` 是否已存在：`grep -rn "func parseUintParam" backend/internal/api/`；若无则加上。）

- [ ] **Step 2: 编译验证**

Run: `go build -C d:\code\xiaoye-ai\backend ./...` → 通过。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/api/novel_handlers.go
git commit -m "feat(novel): storyboard trigger/get/edit handlers with credits"
```

---

## Task 8: 注册路由

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: 在 `/api` 组里加 novel 路由**

在 `main.go` 的 `apiGroup` 内（建议放在 `generationsGroup` 之后、`adminGroup` 之前）加：
```go
		// Novel
		novelGroup := apiGroup.Group("/novel")
		novelGroup.Use(api.UserAuthMiddleware())
		{
			novelGroup.POST("/upload", api.UploadNovel)
			novelGroup.GET("", api.ListNovels)
			novelGroup.GET("/:id", api.GetNovel)
			novelGroup.DELETE("/:id", api.DeleteNovel)
			novelGroup.POST("/chapter/:id/storyboard", api.TriggerStoryboard)
			novelGroup.GET("/chapter/:id/storyboard", api.GetStoryboard)
			novelGroup.PUT("/shot/:id", api.UpdateShot)
		}
```

> 注意：Gin 路由 `/novel/:id` 与 `/novel/chapter/:id/...`、`/novel/shot/:id` 会冲突（同一段位置 `:id` 与 `chapter` 冲突）。解决办法：把 chapter/shot 路径改为参数不冲突的形式——用 `/novel/chapters/:id/storyboard`（复数 chapters）与 `/novel/shots/:id`，与 `/:id` 区分。**采用此修正**：
```go
		novelGroup := apiGroup.Group("/novel")
		novelGroup.Use(api.UserAuthMiddleware())
		{
			novelGroup.POST("/upload", api.UploadNovel)
			novelGroup.GET("", api.ListNovels)
			novelGroup.GET("/:id", api.GetNovel)
			novelGroup.DELETE("/:id", api.DeleteNovel)
			novelGroup.POST("/chapters/:id/storyboard", api.TriggerStoryboard)
			novelGroup.GET("/chapters/:id/storyboard", api.GetStoryboard)
			novelGroup.PUT("/shots/:id", api.UpdateShot)
		}
```
（spec 里写的是 `/novel/chapter/:id`，实现用 `/novel/chapters/:id` 复数以避免 Gin 路由冲突——这是实现细节修正，行为一致。）

- [ ] **Step 2: 重启后端，确认 listening**

Run: 重启后端，日志 `server listening on :8092`，无 panic。

- [ ] **Step 3: 端到端冒烟（curl）**

```bash
# 1. 登录拿 token（video@test.local / Test1234!）
# 2. 上传一个测试 txt（含"第一章"/"第二章"）
curl -X POST http://localhost:8092/api/novel/upload \
  -H "Authorization: Bearer <TOKEN>" -F "file=@test.txt"
# 期望: {novel_id, title, chapter_count:2}
# 3. GET /api/novel → 列表
# 4. GET /api/novel/<id> → 章节
# 5. POST /api/novel/chapters/<chid>/storyboard → {status:extracting}
# 6. 轮询 GET /api/novel/chapters/<chid>/storyboard → status:ready, 9 shots
# 7. PUT /api/novel/shots/<shotid> -d '{"prompt":"改后的提示词"}'
```
Expected: 全链路通，9 个分镜字段齐全。

- [ ] **Step 4: Commit**

```bash
git add backend/main.go
git commit -m "feat(novel): register /api/novel routes"
```

---

## Task 9: 前端 store + composable

**Files:**
- Create: `frontend/src/composables/useNovel.js`
- Create: `frontend/src/stores/novel.js`

- [ ] **Step 1: composable（API 封装）**

`frontend/src/composables/useNovel.js`:
```js
import axios from 'axios'

const instance = axios.create({ baseURL: '/api/novel' })
instance.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export function useNovel() {
  const authHeaders = () => {
    const token = localStorage.getItem('token')
    return token ? { Authorization: `Bearer ${token}` } : {}
  }
  return {
    uploadNovel: (file) => {
      const fd = new FormData()
      fd.append('file', file)
      return instance.post('/upload', fd, { headers: { ...authHeaders(), 'Content-Type': 'multipart/form-data' }, timeout: 60000 })
    },
    listNovels: (params) => instance.get('', { params }),
    getNovel: (id) => instance.get(`/${id}`),
    deleteNovel: (id) => instance.delete(`/${id}`),
    triggerStoryboard: (chapterId) => instance.post(`/chapters/${chapterId}/storyboard`),
    getStoryboard: (chapterId) => instance.get(`/chapters/${chapterId}/storyboard`),
    updateShot: (shotId, payload) => instance.put(`/shots/${shotId}`, payload),
  }
}
```

- [ ] **Step 2: store**

`frontend/src/stores/novel.js`:
```js
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useNovel } from '../composables/useNovel'

export const useNovelStore = defineStore('novel', () => {
  const api = useNovel()
  const novels = ref([])
  const currentNovel = ref(null)
  const chapters = ref([])
  const currentStoryboard = ref(null) // {status, shots, title, content}

  async function loadNovels() {
    const { data } = await api.listNovels({ limit: 50 })
    novels.value = data.items
    return data
  }
  async function uploadNovel(file) {
    const { data } = await api.uploadNovel(file)
    await loadNovels()
    return data
  }
  async function openNovel(id) {
    const { data } = await api.getNovel(id)
    currentNovel.value = data.novel
    chapters.value = data.chapters
    return data
  }
  async function removeNovel(id) {
    await api.deleteNovel(id)
    await loadNovels()
  }
  async function loadStoryboard(chapterId) {
    const { data } = await api.getStoryboard(chapterId)
    currentStoryboard.value = data
    return data
  }
  async function triggerStoryboard(chapterId) {
    await api.triggerStoryboard(chapterId)
  }
  async function saveShot(shotId, payload) {
    await api.updateShot(shotId, payload)
  }
  return { novels, currentNovel, chapters, currentStoryboard,
    loadNovels, uploadNovel, openNovel, removeNovel, loadStoryboard, triggerStoryboard, saveShot }
})
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/composables/useNovel.js frontend/src/stores/novel.js
git commit -m "feat(novel): frontend store + composable"
```

---

## Task 10: NovelList 页面

**Files:**
- Create: `frontend/src/views/NovelList.vue`

- [ ] **Step 1: 实现 NovelList.vue**

```vue
<template>
  <div class="novel-list">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
      <h2>{{ t('novel.title') }}</h2>
      <n-upload :show-file-list="false" accept=".txt" :custom-request="handleUpload">
        <n-button type="primary" :loading="uploading">{{ t('novel.upload') }}</n-button>
      </n-upload>
    </div>
    <n-list v-if="store.novels.length">
      <n-list-item v-for="n in store.novels" :key="n.id">
        <n-thing :title="n.title" :description="`${n.chapter_count} 章 · ${new Date(n.created_at).toLocaleString()}`">
          <template #action>
            <n-button size="small" @click="router.push({name:'novel-detail',params:{id:n.id}})">{{ t('novel.open') }}</n-button>
            <n-popconfirm @positive-click="store.removeNovel(n.id)">
              <template #trigger><n-button size="small" quaternary type="error">{{ t('novel.delete') }}</n-button></template>
              {{ t('novel.deleteConfirm') }}
            </n-popconfirm>
          </template>
        </n-thing>
      </n-list-item>
    </n-list>
    <n-empty v-else :description="t('novel.empty')" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '../stores/novel'
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const router = useRouter()
const store = useNovelStore()
const user = useUserStore()
const uploading = ref(false)

onMounted(async () => {
  if (!user.isLoggedIn) { user.openAuth(); return }
  await store.loadNovels()
})

async function handleUpload({ file }) {
  uploading.value = true
  try {
    const res = await store.uploadNovel(file.file)
    if (res.warning) window.$message?.warning(res.warning)
    window.$message?.success('上传成功')
    router.push({ name: 'novel-detail', params: { id: res.novel_id } })
  } catch (e) {
    window.$message?.error(e.response?.data?.error || '上传失败')
  } finally {
    uploading.value = false
  }
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/NovelList.vue
git commit -m "feat(novel): NovelList page"
```

---

## Task 11: NovelDetail 页面（章节列表）

**Files:**
- Create: `frontend/src/views/NovelDetail.vue`

- [ ] **Step 1: 实现 NovelDetail.vue**

```vue
<template>
  <div class="novel-detail">
    <n-page-header @back="router.push({name:'novel-list'})">
      <template #title>{{ store.currentNovel?.title || '...' }}</template>
    </n-page-header>
    <n-list style="margin-top:16px">
      <n-list-item v-for="ch in store.chapters" :key="ch.id">
        <n-thing :title="`第${ch.chapter_index}章 · ${ch.title || ''}`">
          <template #description>
            <n-tag :type="statusType(ch.storyboard_status)" size="small">{{ statusText(ch.storyboard_status) }}</n-tag>
          </template>
          <template #action>
            <n-button size="small" @click="router.push({name:'chapter-storyboard',params:{cid:ch.id}})">{{ t('novel.storyboard') }}</n-button>
          </template>
        </n-thing>
      </n-list-item>
    </n-list>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

onMounted(() => store.openNovel(route.params.id))

function statusType(s) { return { none: 'default', extracting: 'info', ready: 'success', failed: 'error' }[s] || 'default' }
function statusText(s) { return { none: '未抽取', extracting: '抽取中', ready: '就绪', failed: '失败' }[s] || s }
</script>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/NovelDetail.vue
git commit -m "feat(novel): NovelDetail chapter list page"
```

---

## Task 12: ChapterStoryboard 页面（9 张可编辑卡片 + 轮询）

**Files:**
- Create: `frontend/src/views/ChapterStoryboard.vue`

- [ ] **Step 1: 实现 ChapterStoryboard.vue**

```vue
<template>
  <div class="chapter-storyboard">
    <n-page-header @back="router.back()">
      <template #title>{{ store.currentStoryboard?.title || '分镜' }}</template>
      <template #extra>
        <n-button type="primary" :loading="status==='extracting'" :disabled="status==='extracting'" @click="onGenerate">{{ t('novel.generateStoryboard') }}</n-button>
      </template>
    </n-page-header>

    <n-spin :show="status === 'extracting'">
      <div class="shots-grid" v-if="shots.length">
        <n-card v-for="s in shots" :key="s.id" :title="`分镜 ${s.shot_index}`" size="small" style="margin:8px">
          <n-space vertical>
            <n-input v-model:value="s.scene" type="textarea" :autosize="{minRows:1}" placeholder="场景" @blur="save(s)" />
            <n-input v-model:value="s.characters" placeholder="人物" @blur="save(s)" />
            <n-input v-model:value="s.prompt" type="textarea" :autosize="{minRows:2}" placeholder="图片提示词" @blur="save(s)" />
            <n-input v-model:value="s.dialogue" type="textarea" :autosize="{minRows:1}" placeholder="对白/旁白" @blur="save(s)" />
            <n-input v-model:value="s.camera" placeholder="镜头运动" @blur="save(s)" />
          </n-space>
        </n-card>
      </div>
      <n-empty v-else-if="status!=='extracting'" :description="t('novel.noStoryboard')" />
    </n-spin>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

const status = ref('none')
let timer = null

const shots = computed(() => store.currentStoryboard?.shots || [])

onMounted(async () => { await poll() })
onUnmounted(() => { if (timer) clearTimeout(timer) })

async function poll() {
  try {
    const data = await store.loadStoryboard(route.params.cid)
    status.value = data.status
    if (status.value === 'extracting') {
      timer = setTimeout(poll, 3000)
    }
  } catch (e) { window.$message?.error('加载失败') }
}

async function onGenerate() {
  try {
    await store.triggerStoryboard(route.params.cid)
    status.value = 'extracting'
    store.currentStoryboard = { ...store.currentStoryboard, shots: [] }
    timer = setTimeout(poll, 3000)
  } catch (e) {
    window.$message?.error(e.response?.data?.error || '抽取失败')
  }
}

let saveTimer = null
function save(s) {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    store.saveShot(s.id, { scene: s.scene, characters: s.characters, prompt: s.prompt, dialogue: s.dialogue, camera: s.camera })
      .catch(() => window.$message?.error('保存失败'))
  }, 400)
}
</script>

<style scoped>
.shots-grid { display:grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); margin-top:16px }
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/ChapterStoryboard.vue
git commit -m "feat(novel): ChapterStoryboard editor with polling"
```

---

## Task 13: 路由 + 导航 + i18n

**Files:**
- Modify: `frontend/src/router/index.js`
- Modify: `frontend/src/components/AppSidebar.vue`
- Modify: `frontend/src/locales/zh.json` / `en.json`

- [ ] **Step 1: 加路由**

`router/index.js` 顶部 lazy import 加：
```js
const NovelList = () => import('../views/NovelList.vue')
const NovelDetail = () => import('../views/NovelDetail.vue')
const ChapterStoryboard = () => import('../views/ChapterStoryboard.vue')
```
在 `AppLayout` 的 `children` 里加：
```js
{ path: 'novel', name: 'novel-list', component: NovelList, meta: { titleKey: 'seo.novel.title', requiresAuth: true } },
{ path: 'novel/:id', name: 'novel-detail', component: NovelDetail, meta: { requiresAuth: true } },
{ path: 'novel/chapter/:cid', name: 'chapter-storyboard', component: ChapterStoryboard, meta: { requiresAuth: true } },
```

- [ ] **Step 2: 加导航项**

`AppSidebar.vue` 的 `navItems` 加：
```js
{ name: 'novel-list', label: t('nav.novel'), icon: 'novel' }
```
（若无 `novel` 图标，复用一个现有的如 `book`。）

- [ ] **Step 3: 加 i18n**

`zh.json` 加：
```json
"nav": { "novel": "小说" },
"novel": {
  "title": "我的小说", "upload": "上传小说", "open": "打开", "delete": "删除",
  "deleteConfirm": "删除整本小说及其章节和分镜？", "empty": "还没有小说，上传一个 .txt 试试",
  "storyboard": "分镜", "generateStoryboard": "生成分镜", "noStoryboard": "点击右上角生成分镜"
},
"seo": { "novel": { "title": "小说分镜 - 小野AI" } }
```
`en.json` 对应英文。

- [ ] **Step 4: 前端冒烟**

浏览器开 http://localhost:5173/novel → 登录 `video@test.local`/`Test1234!` → 上传 .txt → 看到小说 → 进章节 → 生成分镜 → 9 张可编辑卡片出现 → 改字段失焦自动保存 → 刷新仍在。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/router/index.js frontend/src/components/AppSidebar.vue frontend/src/locales/zh.json frontend/src/locales/en.json
git commit -m "feat(novel): routes, nav, i18n"
```

---

## Task 14: 端到端验收

按 spec §11 验收标准逐条验证：

- [ ] 上传含「第X章」的 .txt → 正确切多章，列表可见
- [ ] 上传无标题 .txt → 整文 1 章 + warning
- [ ] 某章生成分镜 → 扣 1 钻 → 9 个分镜（5 字段齐全）
- [ ] 编辑分镜字段 → 保存 → 刷新仍在
- [ ] 抽取失败（断中转站）→ status=failed + 钻退回 + 可重试
- [ ] 越权访问 → 403

（全部通过后，本子项目完成。后续 B-2 生图、B-3 视频、A 资源库另开 spec。）
