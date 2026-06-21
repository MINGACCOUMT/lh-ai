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
