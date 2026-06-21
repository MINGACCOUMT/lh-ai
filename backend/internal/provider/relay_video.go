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

// RelayVideoProvider 经 OpenAI 兼容中转站生成视频（ai-tudou: POST /v1/videos + GET /v1/videos/{id}）。
// 复用 OPENAI_BASE_URL / OPENAI_API_KEY。配置后接管 Veo / Seedance 模型 ID；
// 未配置时回退到原直连 provider（google / volcengine）。
//
// 每个实例绑定一个项目模型 ID 及其在中转站的模型名、计费表（不同模型计费不同，故按实例配置）。
type RelayVideoProvider struct {
	projectModelID  string            // 项目侧模型 ID（前端/handler 用）
	relayModel      string            // 中转站侧模型名
	providerName    string            // 注册表名
	displayName     string
	creditsPerSec   map[string]int    // resolution -> credits/秒
	audioMultiplier float64           // Veo 原生含音频=1.0；Seedance 生音频 ×1.2
}

// origVideoProviderName 记录模型 ID 原本指向的 provider 名（中转未配置时回退用）
var origVideoProviderName = map[string]string{}

func init() {
	providers := []*RelayVideoProvider{
		{
			projectModelID:  "veo-3.1-generate-preview",
			relayModel:      "veo_3_1",
			providerName:    "relay-veo",
			displayName:     "Veo 3.1 (中转)",
			creditsPerSec:   VeoCreditsPerSecond, // {720p:45,1080p:65,4k:90}
			audioMultiplier: 1.0,
		},
		{
			projectModelID:  "doubao-seedance-1-5-pro-251215",
			relayModel:      "doubao-seedance-1-5-pro-251215",
			providerName:    "relay-seedance",
			displayName:     "Seedance-1.5 (中转)",
			creditsPerSec:   map[string]int{"480p": 6, "720p": 10, "1080p": 16},
			audioMultiplier: 1.2,
		},
		{
			projectModelID:  "sora-2",
			relayModel:      "sora-2",
			providerName:    "relay-sora",
			displayName:     "Sora 2 (中转)",
			creditsPerSec:   map[string]int{"480p": 20, "720p": 30, "1080p": 50},
			audioMultiplier: 1.0,
		},
	}
	for _, p := range providers {
		RegisterVideoProvider(p.providerName, p)
		// 保存原映射（var 初始化已填好默认值），再覆盖为中转
		origVideoProviderName[p.projectModelID] = modelProviderMap[p.projectModelID]
		modelProviderMap[p.projectModelID] = p.providerName
	}
	log.Printf("[RelayVideo] 中转站视频 provider 已注册（配置 OPENAI_BASE_URL 后生效，否则回退直连）")
}

func (r *RelayVideoProvider) GetProviderName() string { return r.providerName }

func (r *RelayVideoProvider) IsAvailable() bool { return relayConfigured() }

func (r *RelayVideoProvider) GetSupportedModels() []VideoModel {
	return []VideoModel{{
		ID:          r.projectModelID,
		Name:        r.displayName,
		Provider:    r.providerName,
		Description: "经 OpenAI 兼容中转站",
	}}
}

func (r *RelayVideoProvider) CalculateCredits(resolution string, duration int, generateAudio bool) int {
	base, ok := r.creditsPerSec[strings.ToLower(resolution)]
	if !ok || base == 0 {
		base = r.creditsPerSec["720p"] // 兜底
	}
	if duration <= 0 {
		duration = 8
	}
	credits := base * duration
	if generateAudio && r.audioMultiplier != 1.0 {
		credits = int(float64(credits) * r.audioMultiplier)
	}
	return credits
}

// delegate 原直连 provider（中转未配置时）
func (r *RelayVideoProvider) original() VideoProvider {
	if name, ok := origVideoProviderName[r.projectModelID]; ok {
		if p := GetVideoProvider(name); p != nil {
			return p
		}
	}
	return nil
}

func (r *RelayVideoProvider) CreateVideoTask(req VideoGenerateRequest) (*VideoTaskResult, error) {
	if !relayConfigured() {
		if p := r.original(); p != nil {
			return p.CreateVideoTask(req)
		}
		return nil, fmt.Errorf("视频服务未配置")
	}

	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	apiKey := os.Getenv("OPENAI_API_KEY")

	body := relayVideoRequest{
		Model:  r.relayModel,
		Prompt: req.Prompt,
		Size:   mapVideoSize(req.Ratio, req.Resolution),
	}
	jsonData, _ := json.Marshal(body)

	apiURL := baseURL + "/v1/videos"
	client := &http.Client{Timeout: 60 * time.Second}
	log.Printf("[RelayVideo] 提交任务: %s model=%s size=%s", apiURL, r.relayModel, body.Size)

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("提交失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("提交失败 (状态码 %d): %s", resp.StatusCode, truncateForLog(string(rb), 300))
	}

	var res relayVideoTaskResponse
	if err := json.Unmarshal(rb, &res); err != nil {
		return nil, fmt.Errorf("响应解析失败: %v", err)
	}
	if res.ID == "" {
		msg := res.Message
		if res.Error != nil && res.Error.Message != "" {
			msg = res.Error.Message
		}
		if msg == "" {
			msg = truncateForLog(string(rb), 200)
		}
		return nil, fmt.Errorf("中转站未返回任务ID: %s", msg)
	}
	log.Printf("[RelayVideo] 任务已创建: %s", res.ID)
	return &VideoTaskResult{TaskID: res.ID, Status: "queued"}, nil
}

func (r *RelayVideoProvider) GetVideoTaskStatus(taskID string) (*VideoTaskStatusResponse, error) {
	if !relayConfigured() {
		if p := r.original(); p != nil {
			return p.GetVideoTaskStatus(taskID)
		}
		return nil, fmt.Errorf("视频服务未配置")
	}

	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	apiKey := os.Getenv("OPENAI_API_KEY")
	apiURL := baseURL + "/v1/videos/" + taskID

	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, _ := http.NewRequest("GET", apiURL, nil)
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("查询失败 (状态码 %d): %s", resp.StatusCode, truncateForLog(string(rb), 300))
	}

	var res relayVideoTaskResponse
	if err := json.Unmarshal(rb, &res); err != nil {
		return nil, fmt.Errorf("响应解析失败: %v", err)
	}

	out := &VideoTaskStatusResponse{TaskID: res.ID, VideoURL: res.VideoURL}
	switch strings.ToLower(res.Status) {
	case "completed", "succeeded", "success":
		out.Status = "succeeded"
	case "failed", "error":
		out.Status = "failed"
		if res.Error != nil && res.Error.Message != "" {
			out.ErrorMessage = res.Error.Message
		} else if res.Message != "" {
			out.ErrorMessage = res.Message
		}
	case "processing", "running", "in_progress":
		out.Status = "running"
	default: // queued / 空 / 未知
		out.Status = "queued"
	}
	return out, nil
}

// mapVideoSize 项目 ratio(16:9 等) + resolution(720p 等) → 中转站 size(WxH)
func mapVideoSize(ratio, resolution string) string {
	r := strings.TrimSpace(ratio)
	hd := strings.EqualFold(resolution, "1080p") || strings.EqualFold(resolution, "4k")
	switch r {
	case "9:16":
		if hd {
			return "1080x1920"
		}
		return "720x1280"
	case "1:1":
		if hd {
			return "1080x1080"
		}
		return "720x720"
	default: // 16:9 及其它默认横屏
		if hd {
			return "1920x1080"
		}
		return "1280x720"
	}
}

// ============ 中转站视频请求/响应结构 ============

type relayVideoRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Size   string   `json:"size,omitempty"`
	Images []string `json:"images,omitempty"` // 参考图/首尾帧 URL（v1 文生视频不传）
}

// 兼容两种返回：成功 {id,status,progress,video_url,error}；错误 {code,message,data}
type relayVideoTaskResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	VideoURL  string `json:"video_url"`
	Error     *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
	// 错误信封
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
