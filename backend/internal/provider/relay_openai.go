package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// RelayImageModel 通过 OpenAI 兼容中转站（relay）生成图像。
//
// 设计：init() 无条件覆盖标准 Gemini 图像模型 ID（gemini-3-pro-image-preview /
// gemini-3.1-flash-image-preview），并保留 gemini.go 注册的原始直连 provider。
// 调用时（GenerateImage/IsAvailable）再判断 OPENAI_BASE_URL 是否配置：
//   - 已配置 → 走 OpenAI /v1/chat/completions 中转站（绕开直连 Google 的配额/封禁）
//   - 未配置 → 回退到 gemini.go 的直连 Google（行为不变）
//
// 注意：Go 的 init() 早于 main()/godotenv，不能在 init 里判 env，故采用"无条件覆盖 + 调用时分流"。
// 文件名 relay_openai.go 排序在 gemini.go 之后，init() 后执行，覆盖 + 保留均生效。
type RelayImageModel struct {
	id   string // 模型 ID（同时作为中转站的 model 名）
	name string // 显示名称
}

// directGeminiImage 保留 gemini.go 注册的直连 provider，中转站未配置时回退使用。
var directGeminiImage = map[string]ImageGenerator{}

var relayModelIDs = []string{"gemini-3-pro-image-preview", "gemini-3.1-flash-image-preview"}

var relayModelNames = map[string]string{
	"gemini-3-pro-image-preview":   "Nanobanana Pro",
	"gemini-3.1-flash-image-preview": "Nanobanana 2",
}

func init() {
	// 先保留 gemini.go 注册的原始直连 provider
	for _, id := range relayModelIDs {
		if g, ok := models[id]; ok {
			directGeminiImage[id] = g
		}
	}
	// 无条件用 RelayImageModel 覆盖（调用时再分流）
	for _, id := range relayModelIDs {
		Register(id, &RelayImageModel{id: id, name: relayModelNames[id]})
	}
	log.Printf("[Relay] OpenAI 兼容中转站 provider 已注册（配置 OPENAI_BASE_URL 后生效，否则回退直连 Google）")
}

func relayConfigured() bool {
	return os.Getenv("OPENAI_BASE_URL") != "" && os.Getenv("OPENAI_API_KEY") != ""
}

func (m *RelayImageModel) ID() string   { return m.id }
func (m *RelayImageModel) Name() string { return m.name }

func (m *RelayImageModel) Provider() string {
	if relayConfigured() {
		return "relay"
	}
	if g, ok := directGeminiImage[m.id]; ok {
		return g.Provider()
	}
	return "relay"
}

func (m *RelayImageModel) IsAvailable() bool {
	if relayConfigured() {
		return true
	}
	if g, ok := directGeminiImage[m.id]; ok {
		return g.IsAvailable()
	}
	return false
}

func (m *RelayImageModel) GenerateImage(prompt string, opts ImageOptions) (*ImageResult, error) {
	if relayConfigured() {
		return m.generateViaRelay(prompt, opts)
	}
	if g, ok := directGeminiImage[m.id]; ok {
		return g.GenerateImage(prompt, opts)
	}
	return nil, fmt.Errorf("图像服务未配置（中转站未配置且直连 Google 不可用）")
}

func (m *RelayImageModel) generateViaRelay(prompt string, opts ImageOptions) (*ImageResult, error) {
	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("中转站未配置 (OPENAI_BASE_URL / OPENAI_API_KEY)")
	}

	// 中转站提供 -2k / -4k 尺寸变体，按需映射
	model := m.id
	switch strings.ToLower(opts.ImageSize) {
	case "2k":
		model = m.id + "-2k"
	case "4k":
		model = m.id + "-4k"
	}

	// OpenAI 协议无原生 aspectRatio，作为文字提示附加
	fullPrompt := prompt
	if ar := strings.TrimSpace(opts.AspectRatio); ar != "" {
		fullPrompt = fmt.Sprintf("%s\n\n(aspect ratio: %s)", prompt, ar)
	}

	// 有输入图时用多模态 content（支持图生图，best-effort）；否则用纯字符串（最稳）
	var content interface{} = fullPrompt
	if len(opts.InputImages) > 0 {
		parts := []relayContentPart{{Type: "text", Text: fullPrompt}}
		for _, img := range opts.InputImages {
			parts = append(parts, relayContentPart{
				Type:     "image_url",
				ImageURL: &relayImageURL{URL: "data:image/png;base64," + img},
			})
		}
		content = parts
	}

	reqBody := relayChatRequest{
		Model:    model,
		Stream:   false,
		Messages: []relayMessage{{Role: "user", Content: content}},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/v1/chat/completions"
	client := &http.Client{Timeout: 300 * time.Second}
	log.Printf("[Relay] 调用中转站生图: %s model=%s", apiURL, model)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Relay] HTTP 请求失败: %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("生成失败 (状态码 %d): %s", resp.StatusCode, truncateForLog(string(body), 300))
	}

	// 校验结构（便于报错），base64 直接从原始响应提取，兼容 markdown 包装 / string / 数组等多种 content 形态
	var chatResp relayChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("中转站响应解析失败: %v", err)
	}
	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return nil, fmt.Errorf("中转站返回错误: %s", chatResp.Error.Message)
	}

	data, mime, ok := findImageBase64(string(body))
	if !ok {
		if len(chatResp.Choices) > 0 {
			return nil, fmt.Errorf("中转站响应中未找到图像数据 (content: %s)", truncateForLog(chatResp.Choices[0].Message.Content, 200))
		}
		return nil, fmt.Errorf("中转站响应中未找到图像数据")
	}

	log.Printf("[Relay] 图像生成成功，%d 字节 base64", len(data))
	return &ImageResult{Data: data, MimeType: mime}, nil
}

// findImageBase64 在响应文本中定位 data:image/xxx;base64,.... 并提取纯净 base64 与 MIME。
// 中转站常见格式: ![image](data:image/png;base64,iVBOR...) 或裸 data: URL。
func findImageBase64(s string) (data, mime string, ok bool) {
	idx := strings.Index(s, "base64,")
	if idx < 0 {
		return "", "", false
	}
	// 反推 MIME（image/png、image/jpeg、image/webp ...）
	mime = "image/png"
	if head := strings.LastIndex(s[:idx], "image/"); head >= 0 {
		rest := s[head+len("image/") : idx]
		if semi := strings.IndexByte(rest, ';'); semi >= 0 {
			rest = rest[:semi]
		}
		rest = strings.TrimSpace(rest)
		if rest != "" {
			mime = "image/" + rest
		}
	}
	start := idx + len("base64,")
	var b strings.Builder
	b.Grow(1 << 16)
	for i := start; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=' {
			b.WriteByte(c)
		} else {
			break
		}
	}
	data = b.String()
	if data == "" {
		return "", "", false
	}
	return data, mime, true
}

func truncateForLog(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ============ OpenAI 兼容请求/响应结构 ============

type relayChatRequest struct {
	Model    string         `json:"model"`
	Stream   bool           `json:"stream"`
	Messages []relayMessage `json:"messages"`
}

type relayMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string 或 []relayContentPart
}

type relayContentPart struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *relayImageURL `json:"image_url,omitempty"`
}

type relayImageURL struct {
	URL string `json:"url"`
}

type relayChatResponse struct {
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}
