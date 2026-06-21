package novel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google-ai-proxy/internal/config"
)

type relayChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RelayChat 调 Anthropic 兼容的 chat 端点（{base}/v1/messages），用于小说分镜/分析。
// base/key/model 由 NOVEL_LLM_* 配置（默认回退 OPENAI_*）。
// 兼容智谱 GLM 的 /api/anthropic 等 Anthropic 协议网关。返回 assistant 文本。
func RelayChat(model, systemPrompt, userContent string) (string, error) {
	baseURL := strings.TrimRight(config.GetNovelLLMBaseURL(), "/")
	apiKey := config.GetNovelLLMAPIKey()
	if baseURL == "" || apiKey == "" {
		return "", fmt.Errorf("小说 LLM 未配置 (NOVEL_LLM_BASE_URL / NOVEL_LLM_API_KEY)")
	}

	// Anthropic Messages API: system 单独字段，messages 不含 system
	body := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096,
		"system":     systemPrompt,
		"messages":   []relayChatMessage{{Role: "user", Content: userContent}},
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL+"/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用失败: %v", err)
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM 返回 %d: %s", resp.StatusCode, truncate(string(rb), 300))
	}

	// Anthropic 响应: content 是 block 数组，取 type=text 的拼接
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return "", fmt.Errorf("响应解析失败: %v", err)
	}
	var sb strings.Builder
	for _, b := range parsed.Content {
		if b.Type == "text" && b.Text != "" {
			sb.WriteString(b.Text)
		}
	}
	if sb.Len() == 0 {
		return "", fmt.Errorf("LLM 未返回文本内容")
	}
	return sb.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
