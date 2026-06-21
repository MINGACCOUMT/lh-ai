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

// Plot 是一章里的一个情节。
type Plot struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// Analysis 是一章的解析结果。
type Analysis struct {
	Outline    string `json:"outline"`
	Characters string `json:"characters"`
	Scenes     string `json:"scenes"`
	Plots      []Plot `json:"plots"`
}

const analyzeSystemPrompt = `你是一名资深小说编辑兼影视策划。我会给你一章小说正文，请分析并返回 JSON：
{"outline":"本章大纲（100~200字，概括主要情节）","characters":[{"name":"人物名","description":"简短画像（外貌/身份/性格）"}],"scenes":[{"name":"场景/地点名","description":"场景描述"}],"plots":[{"title":"情节标题","summary":"该情节简述（50~100字）"}]}
要求：characters 是本章出场人物数组；scenes 是关键场景数组；plots 按本章主要情节切分（通常 2~5 个）覆盖整章脉络。只返回 JSON，不要解释。`

// AnalyzeChapter 解析一章：大纲 + 人物画像 + 场景 + 情节列表。
func AnalyzeChapter(chapterContent string) (*Analysis, error) {
	maxChars := config.GetNovelChapterMaxChars()
	content := chapterContent
	if len([]rune(content)) > maxChars {
		content = string([]rune(content)[:maxChars])
	}
	model := config.GetNovelLLMModel()
	raw, err := RelayChat(model, analyzeSystemPrompt, content)
	if err != nil {
		return nil, err
	}
	return parseAnalysisJSON(raw)
}

// parseAnalysisJSON 解析 LLM 返回的 Analysis（兼容裸 JSON / ```json 代码块）。
// characters/scenes 用 RawMessage 灵活接收（LLM 可能返回字符串或数组），再归一化成文本。
func parseAnalysisJSON(raw string) (*Analysis, error) {
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
	var flex struct {
		Outline    string          `json:"outline"`
		Characters json.RawMessage `json:"characters"`
		Scenes     json.RawMessage `json:"scenes"`
		Plots      []Plot          `json:"plots"`
	}
	if err := json.Unmarshal([]byte(raw), &flex); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %v (raw: %s)", err, truncate(raw, 200))
	}
	if len(flex.Plots) == 0 {
		return nil, fmt.Errorf("LLM 未返回情节列表")
	}
	return &Analysis{
		Outline:    flex.Outline,
		Characters: normalizeTextField(flex.Characters),
		Scenes:     normalizeTextField(flex.Scenes),
		Plots:      flex.Plots,
	}, nil
}

// normalizeTextField 把 LLM 返回的人物/场景字段归一化成可读多行文本。
// 兼容：字符串 / 字符串数组 / {name,description} 对象数组 / 其它（原样 JSON）。
func normalizeTextField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var arr []interface{}
	if json.Unmarshal(raw, &arr) == nil {
		var lines []string
		for _, e := range arr {
			lines = append(lines, normalizeEntry(e))
		}
		return strings.Join(lines, "\n")
	}
	return strings.TrimSpace(string(raw))
}

func normalizeEntry(e interface{}) string {
	if m, ok := e.(map[string]interface{}); ok {
		name := mapStr(m, "name")
		desc := mapStr(m, "description")
		if desc == "" {
			desc = mapStr(m, "desc")
		}
		if name != "" {
			return name + "：" + desc
		}
		b, _ := json.Marshal(m)
		return string(b)
	}
	return fmt.Sprintf("%v", e)
}

func mapStr(m map[string]interface{}, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
