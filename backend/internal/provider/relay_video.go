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

	"google-ai-proxy/internal/storage"
)

// RelayVideoProvider 经 OpenAI 兼容中转站生成视频。
// 复用 OPENAI_BASE_URL / OPENAI_API_KEY。配置后接管 Veo / Seedance / Sora 模型 ID；
// 未配置时回退到原直连 provider（google / volcengine）。
//
// 每个实例绑定一个项目模型 ID 及其在中转站的模型名、计费表（不同模型计费不同，故按实例配置）。
//
// apiType 区分两套不同的中转 API：
//   - "veo_new"   : Veo 3.1 新接口 POST /v1/videos/generations + GET /v1/tasks/{id}
//     模型名 veo3.1-720p / veo3.1-1080p 由 resolution 动态计算
//   - "relay_old" : ai-tudou 旧接口 POST /v1/videos + GET /v1/videos/{id}（Seedance / Sora）
type RelayVideoProvider struct {
	projectModelID  string // 项目侧模型 ID（前端/handler 用）
	relayModel      string // 中转站侧模型名（veo_new 下由 resolution 动态计算，留空）
	apiType         string // "veo_new" / "relay_old"
	providerName    string // 注册表名
	displayName     string
	creditsPerSec   map[string]int // resolution -> credits/秒
	audioMultiplier float64        // Veo 原生含音频=1.0；Seedance 生音频 ×1.2
}

// origVideoProviderName 记录模型 ID 原本指向的 provider 名（中转未配置时回退用）
var origVideoProviderName = map[string]string{}

func init() {
	providers := []*RelayVideoProvider{
		{
			projectModelID:  "veo-3.1-generate-preview",
			relayModel:      "", // veo_new 下按 resolution 动态计算
			apiType:         "veo_new",
			providerName:    "relay-veo",
			displayName:     "Veo 3.1 (中转)",
			creditsPerSec:   VeoCreditsPerSecond, // {720p:45,1080p:65,4k:90}
			audioMultiplier: 1.0,
		},
		{
			projectModelID:  "doubao-seedance-1-5-pro-251215",
			relayModel:      "doubao-seedance-1-5-pro-251215",
			apiType:         "relay_old",
			providerName:    "relay-seedance",
			displayName:     "Seedance-1.5 (中转)",
			creditsPerSec:   map[string]int{"480p": 6, "720p": 10, "1080p": 16},
			audioMultiplier: 1.2,
		},
		{
			projectModelID:  "sora-2",
			relayModel:      "sora-2",
			apiType:         "relay_old",
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
	if r.apiType == "veo_new" {
		return r.createVideoTaskVeo(req)
	}
	return r.createVideoTaskOld(req)
}

func (r *RelayVideoProvider) GetVideoTaskStatus(taskID string) (*VideoTaskStatusResponse, error) {
	if !relayConfigured() {
		if p := r.original(); p != nil {
			return p.GetVideoTaskStatus(taskID)
		}
		return nil, fmt.Errorf("视频服务未配置")
	}
	if r.apiType == "veo_new" {
		return r.getVideoTaskStatusVeo(taskID)
	}
	return r.getVideoTaskStatusOld(taskID)
}

// ============================ Veo 3.1 新接口 ============================

// veoRelayModel 按 resolution 映射到中转站 Veo 3.1 模型名
func veoRelayModel(resolution string) string {
	if strings.Contains(strings.ToLower(resolution), "1080") {
		return "veo3.1-1080p"
	}
	return "veo3.1-720p" // 默认
}

// clampDuration 将时长收敛到 Veo 支持的 4 / 6 / 8 秒
func clampDuration(d int) int {
	if d <= 4 {
		return 4
	}
	if d <= 6 {
		return 6
	}
	return 8
}

// collectImages 汇总首帧 / 尾帧 / 参考图 base64（用于 i2v）
func collectImages(req VideoGenerateRequest) []string {
	var imgs []string
	if req.FirstFrame != "" {
		imgs = append(imgs, req.FirstFrame)
	}
	if req.LastFrame != "" {
		imgs = append(imgs, req.LastFrame)
	}
	imgs = append(imgs, req.ReferenceImages...)
	return imgs
}

// stripBase64Prefix 去掉 data:image/...;base64, 前缀，OSS 上传需要纯 base64
func stripBase64Prefix(s string) string {
	if i := strings.Index(s, "base64,"); i >= 0 {
		return s[i+len("base64,"):]
	}
	return s
}

func (r *RelayVideoProvider) createVideoTaskVeo(req VideoGenerateRequest) (*VideoTaskResult, error) {
	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	apiKey := os.Getenv("OPENAI_API_KEY")

	resolution := strings.ToLower(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		resolution = "720p"
	}
	relayModel := veoRelayModel(resolution)
	duration := clampDuration(req.Duration)

	// base64 → OSS → URL（失败跳过，避免单张失败阻塞整单）
	var imageUrls []string
	for _, img := range collectImages(req) {
		url, err := storage.UploadBase64Image(stripBase64Prefix(img), "veo-ref", "veo-ref")
		if err != nil {
			log.Printf("[RelayVideo][Veo] 上传参考图失败(跳过): %v", err)
			continue
		}
		imageUrls = append(imageUrls, url)
	}

	// reference_mode: frame(1-2 张, 首尾帧) / image(>=3 张, 风格参考)
	referenceMode := "frame"
	if len(imageUrls) >= 3 {
		referenceMode = "image"
		duration = 8 // image 模式仅支持 8 秒
	}

	body := map[string]interface{}{
		"model":          relayModel,
		"prompt":         req.Prompt,
		"resolution":     resolution,
		"duration":       duration,
		"aspect_ratio":   req.Ratio,
		"generate_audio": req.GenerateAudio,
	}
	if len(imageUrls) > 0 {
		body["images"] = imageUrls
		body["reference_mode"] = referenceMode
	}

	jsonData, _ := json.Marshal(body)
	apiURL := baseURL + "/v1/videos/generations"
	client := &http.Client{Timeout: 60 * time.Second}
	log.Printf("[RelayVideo][Veo] 提交任务: %s model=%s resolution=%s duration=%d ratio=%s imgs=%d mode=%s",
		apiURL, relayModel, resolution, duration, req.Ratio, len(imageUrls), referenceMode)

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

	var res veoSubmitResponse
	if err := json.Unmarshal(rb, &res); err != nil {
		return nil, fmt.Errorf("响应解析失败: %v", err)
	}
	if res.Data.ID == "" {
		msg := res.Message
		if res.Error != nil && res.Error.Message != "" {
			msg = res.Error.Message
		}
		if msg == "" {
			msg = truncateForLog(string(rb), 200)
		}
		return nil, fmt.Errorf("中转站未返回任务ID: %s", msg)
	}
	log.Printf("[RelayVideo][Veo] 任务已创建: %s", res.Data.ID)
	return &VideoTaskResult{TaskID: res.Data.ID, Status: "queued"}, nil
}

func (r *RelayVideoProvider) getVideoTaskStatusVeo(taskID string) (*VideoTaskStatusResponse, error) {
	baseURL := strings.TrimRight(os.Getenv("OPENAI_BASE_URL"), "/")
	apiKey := os.Getenv("OPENAI_API_KEY")
	apiURL := baseURL + "/v1/tasks/" + taskID

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

	var res veoQueryResponse
	if err := json.Unmarshal(rb, &res); err != nil {
		return nil, fmt.Errorf("响应解析失败: %v", err)
	}

	out := &VideoTaskStatusResponse{TaskID: taskID, VideoURL: res.Data.Result.VideoURL}
	switch strings.ToLower(strings.TrimSpace(res.Data.Status)) {
	case "completed", "succeeded", "success":
		out.Status = "succeeded"
	case "failed", "error":
		out.Status = "failed"
		if res.Data.Error != nil && res.Data.Error.Message != "" {
			out.ErrorMessage = res.Data.Error.Message
		} else if res.Message != "" {
			out.ErrorMessage = res.Message
		}
	case "processing", "running", "in_progress":
		out.Status = "running"
	default: // submitted / queued / 空 / 未知
		out.Status = "queued"
	}
	return out, nil
}

// ============================ Seedance / Sora 旧接口 ============================

func (r *RelayVideoProvider) createVideoTaskOld(req VideoGenerateRequest) (*VideoTaskResult, error) {
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

func (r *RelayVideoProvider) getVideoTaskStatusOld(taskID string) (*VideoTaskStatusResponse, error) {
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

// ============ 中转站请求/响应结构 ============

// 旧接口请求（Seedance / Sora）
type relayVideoRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Size   string   `json:"size,omitempty"`
	Images []string `json:"images,omitempty"`
}

// 旧接口响应：兼容两种返回（成功 {id,status,progress,video_url,error}；错误 {code,message,data}）
type relayVideoTaskResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	VideoURL string `json:"video_url"`
	Error    *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
	// 错误信封
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// ============ Veo 3.1 新接口响应结构 ============

// veoSubmitResponse 新接口提交响应 {code,message,data:{id},error?}
type veoSubmitResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ID string `json:"id"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// veoQueryResponse 新接口查询响应 {code,data:{status,progress,result:{video_url},error},message?}
type veoQueryResponse struct {
	Code int `json:"code"`
	Data struct {
		Status   string `json:"status"`
		Progress int    `json:"progress"`
		Result   struct {
			VideoURL string `json:"video_url"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
	Message string `json:"message,omitempty"`
}
