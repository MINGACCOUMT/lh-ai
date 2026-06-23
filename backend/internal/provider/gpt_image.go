package provider

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"google-ai-proxy/internal/storage"
	"time"
)

// GPTImageModel 通过 OpenAI 兼容 /v1/images/generations 接口生成图像（如 gpt-image-2）。
// 独立配置 GPT_IMAGE_BASE_URL + GPT_IMAGE_API_KEY（指向专用图像中转，如 koramkoin），
// 模型名由 GPT_IMAGE_MODEL 指定（默认 gpt-image-2）。不干扰 relay_openai.go 的视频/Gemini 图像中转。
type GPTImageModel struct{}

func init() {
	Register("gpt-image-2", &GPTImageModel{})
	log.Printf("[GPTImage] gpt-image-2 provider 已注册（配置 GPT_IMAGE_BASE_URL 后生效）")
}

func gptImageConfigured() bool {
	return os.Getenv("GPT_IMAGE_BASE_URL") != "" && os.Getenv("GPT_IMAGE_API_KEY") != ""
}

func (g *GPTImageModel) ID() string       { return "gpt-image-2" }
func (g *GPTImageModel) Name() string     { return "GPT Image 2" }
func (g *GPTImageModel) Provider() string { return "gpt-image" }
func (g *GPTImageModel) IsAvailable() bool { return gptImageConfigured() }

func (g *GPTImageModel) GenerateImage(prompt string, opts ImageOptions) (*ImageResult, error) {
	baseURL := strings.TrimRight(os.Getenv("GPT_IMAGE_BASE_URL"), "/")
	apiKey := os.Getenv("GPT_IMAGE_API_KEY")
	model := os.Getenv("GPT_IMAGE_MODEL")
	if model == "" {
		model = "gpt-image-2"
	}
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("gpt-image-2 未配置 (GPT_IMAGE_BASE_URL / GPT_IMAGE_API_KEY)")
	}

	// OpenAI 协议无原生 aspectRatio，作为文字提示附加
	fullPrompt := prompt
	if ar := strings.TrimSpace(opts.AspectRatio); ar != "" {
		fullPrompt = fmt.Sprintf("%s\n\n(aspect ratio: %s)", prompt, ar)
	}

	client := &http.Client{Timeout: 180 * time.Second} // 单张超 3 分钟判失败

	// 有参考图 → /v1/images/edits（图生图/编辑）；无参考图 → /v1/images/generations（文生图）
	var apiURL string
	var jsonData []byte

	if len(opts.InputImages) > 0 {
		// 图生图：上传参考图到 OSS → 拿 URL → 调 /v1/images/edits
		apiURL = baseURL + "/v1/images/edits"
		imgURL, err := storage.UploadBase64Image(opts.InputImages[0], "gpt-edit", "gpt-edit")
		if err != nil {
			return nil, fmt.Errorf("上传参考图失败: %v", err)
		}
		editBody := map[string]interface{}{
			"model":  model,
			"prompt": fullPrompt,
			"image":  imgURL,
			"n":      1,
		}
		// mask（局部重绘）
		if opts.MaskImage != "" {
			maskURL, err := storage.UploadBase64Image(opts.MaskImage, "gpt-edit", "gpt-edit")
			if err == nil {
				editBody["mask"] = maskURL
			}
		}
		jsonData, _ = json.Marshal(editBody)
		log.Printf("[GPTImage] 调用 edits: %s model=%s img=%s", apiURL, model, imgURL[:min(60, len(imgURL))])
	} else {
		// 文生图
		apiURL = baseURL + "/v1/images/generations"
		reqBody := gptImageRequest{
			Model:  model,
			Prompt: fullPrompt,
			N:      1,
		}
		jsonData, _ = json.Marshal(reqBody)
		log.Printf("[GPTImage] 调用: %s model=%s", apiURL, model)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[GPTImage] HTTP 请求失败: %v", err)
		return nil, fmt.Errorf("生成失败: %s", sanitizeHTTPError(err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("生成失败 (状态码 %d): %s", resp.StatusCode, truncateForLog(string(body), 300))
	}

	var imgResp gptImageResponse
	if err := json.Unmarshal(body, &imgResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %v", err)
	}
	if imgResp.Error != nil && imgResp.Error.Message != "" {
		return nil, fmt.Errorf("中转站返回错误: %s", imgResp.Error.Message)
	}
	if len(imgResp.Data) == 0 {
		return nil, fmt.Errorf("响应中没有图像数据")
	}

	item := imgResp.Data[0]
	// 优先用 b64_json（项目统一以 base64 上传 OSS）
	if item.B64JSON != "" {
		log.Printf("[GPTImage] 生成成功 (b64_json, %d 字节)", len(item.B64JSON))
		return &ImageResult{Data: item.B64JSON, MimeType: "image/png"}, nil
	}
	// 退化：只有 url 时下载转 base64
	if item.URL != "" {
		b64, mime, err := downloadAsBase64(item.URL)
		if err != nil {
			return nil, fmt.Errorf("下载生成图失败: %v", err)
		}
		log.Printf("[GPTImage] 生成成功 (url->b64, %d 字节)", len(b64))
		return &ImageResult{Data: b64, MimeType: mime}, nil
	}
	return nil, fmt.Errorf("响应中未找到 b64_json 或 url")
}

// downloadAsBase64 下载图片 URL 转 base64（url 退化路径用）
func downloadAsBase64(imageURL string) (string, string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/png"
	}
	return base64.StdEncoding.EncodeToString(data), mime, nil
}

// ============ OpenAI images/generations 结构 ============

type gptImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size,omitempty"`
}

type gptImageResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		B64JSON string `json:"b64_json,omitempty"`
		URL     string `json:"url,omitempty"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}
