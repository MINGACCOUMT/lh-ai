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

// RelayChat 调中转站 chat/completions（OPENAI_BASE_URL），返回 assistant 文本。
// response_format=json_object，要求模型返回 JSON 文本。
func RelayChat(model, systemPrompt, userContent string) (string, error) {
	baseURL := strings.TrimRight(config.GetNovelLLMBaseURL(), "/")
	apiKey := config.GetNovelLLMAPIKey()
	if baseURL == "" || apiKey == "" {
		return "", fmt.Errorf("中转站未配置 (NOVEL_LLM_BASE_URL/API_KEY 或 OPENAI_BASE_URL/API_KEY)")
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
