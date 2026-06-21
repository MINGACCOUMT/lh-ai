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

const storyboardSystemPrompt = `你是一名专业的影视分镜师。我会给你一章小说正文，请先给出本章的简短大纲（100~200字，概括主要情节），然后把它拆成恰好 9 个分镜（镜头）。
严格返回 JSON，格式为：{"outline":"本章大纲概述...","shots":[{"scene":"场景描述","characters":"出场人物，逗号分隔","prompt":"用于 AI 绘图的画面提示词（中文，描述这一镜的视觉画面，不含对白）","dialogue":"该镜头的重要对白或旁白，没有则空字符串","camera":"镜头运动，如 推进/平移/特写/全景/跟随"}, ...共9个...]}
只返回 JSON，不要任何解释。shots 必须恰好 9 个。`

// StoryboardResult 是一章的分镜抽取结果。
type StoryboardResult struct {
	Outline string `json:"outline"`
	Shots   []Shot `json:"shots"`
}

// ExtractStoryboard 调 LLM 抽取一章的大纲 + 9 个分镜。
func ExtractStoryboard(chapterContent string) (*StoryboardResult, error) {
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
	return parseStoryboardJSON(raw)
}

var jsonShotRe = regexp.MustCompile(`(?s)\{.*\}`)

// parseStoryboardJSON 解析 LLM 返回（兼容裸 JSON / ```json 代码块）。shots 超 9 截断、为 0 报错。
func parseStoryboardJSON(raw string) (*StoryboardResult, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}
	if !strings.HasPrefix(raw, "{") {
		if m := jsonShotRe.FindString(raw); m != "" {
			raw = m
		}
	}
	var res StoryboardResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("分镜 JSON 解析失败: %v (raw: %s)", err, truncate(raw, 200))
	}
	if len(res.Shots) == 0 {
		return nil, fmt.Errorf("LLM 未返回任何分镜")
	}
	if len(res.Shots) > 9 {
		res.Shots = res.Shots[:9]
	}
	return &res, nil
}
