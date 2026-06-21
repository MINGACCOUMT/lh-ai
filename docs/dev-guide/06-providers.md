# 06 · AI 供应商适配层

> 目录：[backend/internal/provider/](../../backend/internal/provider/)。这是接新模型/换生成引擎时最该读的部分。

## 设计模式：注册表（Registry） + 接口

供应商通过**自注册**（`init()` 调 `Register/RegisterVideoProvider`）挂到全局 map，handler 通过 ID 查找，**完全解耦**。

### 图像：`ImageGenerator` 接口
定义在 [interface.go](../../backend/internal/provider/interface.go)：
```go
type ImageGenerator interface {
    GenerateImage(prompt string, opts ImageOptions) (*ImageResult, error)
    Name() string
    ID() string
    Provider() string
    IsAvailable() bool  // API Key 是否配置
}
// 扩展接口（电商组图）
type MultiImageGenerator interface {
    ImageGenerator
    GenerateMultiImage(prompt string, inputImages []string, outputCount int, opts ImageOptions) (*MultiImageResult, error)
    SupportsMultiImage() bool
}
```
- `Register(id, generator)` — 自注册
- `Get(id)` — 取模型（不可用报错）
- `GetDefault()` — 按优先级 `['gemini-3-pro-image-preview','doubao-seedream-4-5']` 回落
- `ListAvailable()` — 按 `modelOrder` 排序列出
- `ImageOptions{ AspectRatio, ImageSize, InputImages[]base64, MaskImage base64 }`

### 视频：`VideoProvider` 接口
定义在 [video_interface.go](../../backend/internal/provider/video_interface.go)：
```go
type VideoProvider interface {
    GetProviderName() string
    IsAvailable() bool
    GetSupportedModels() []VideoModel
    CreateVideoTask(req VideoGenerateRequest) (*VideoTaskResult, error)
    GetVideoTaskStatus(taskID string) (*VideoTaskStatusResponse, error)
    CalculateCredits(resolution string, duration int, generateAudio bool) int
}
```
- `RegisterVideoProvider(name, provider)` — 注册服务商
- `modelProviderMap` — 模型ID → 服务商名映射（`GetProviderByModel`）
- `GetVideoProviderForModel(modelID)` — 按模型取服务商

> **图像按模型ID注册，视频按服务商注册 + 模型映射**——两套机制略有差异，接新视频模型要同时更新 `modelProviderMap`。

## 已接入的供应商

### Gemini（图像）— [gemini.go](../../backend/internal/provider/gemini.go)
- 注册模型：`gemini-3-pro-image-preview`(Nanobanana Pro)、`gemini-3.1-flash-image-preview`(Nanobanana 2)
- 可用性门控：`GOOGLE_API_KEY`
- 端点：`https://generativelanguage.googleapis.com/v1beta/models/<model>:generateContent?key=<key>`
- 参数：aspectRatio 透传；imageSize 空→1K，`0.5K`→512px；InputImages 作 inlineData（image/png）；MaskImage 追加为额外 inlineData
- 结果：取 `Candidates[0].Content.Parts` 最后一个非空 inlineData
- **支持 HTTP_PROXY**，超时 300s，无重试，不实现 MultiImage

### 火山引擎 Seedream（图像+多图）— [volcengine.go](../../backend/internal/provider/volcengine.go)
- 注册模型：`doubao-seedream-4-5`(Seedream-4.5，endpoint `doubao-seedream-4-5-251128`)
- 可用性门控：`ARK_API_KEY`
- 端点：`https://ark.cn-beijing.volces.com/api/v3/images/generations`，`Authorization: Bearer`
- **唯一实现 `MultiImageGenerator`**（`supportsMulti=true`, `defaultMaxImages=7`）
  - 多图：输入 1~14 张，输出数 `min(15-输入数, defaultMaxImages)`，`sequential_image_generation=auto`，超时 600s
- 尺寸：`convertSize` 仅 `4K→4K` 其余→`2K`；宽高比转成中文自然语言后缀追加到 prompt（如 `16:9` → `，横版16:9宽屏比例`）
- **不支持 HTTP_PROXY**，无重试

### 火山引擎 Seedance（视频）— [volcengine_video.go](../../backend/internal/provider/volcengine_video.go)
- 注册：`RegisterVideoProvider("volcengine", ...)`，模型 `doubao-seedance-1-5-pro-251215`(Seedance-1.5)
- 端点：`https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks`（POST 创建 / GET +`/<taskID>` 查询）
- 模式：first-frame（仅首帧）/ first-last-frame（首+尾帧），图片 `data:image/png;base64,` 前缀
- 约束：duration `<4→5`、`>12→12`；resolution 空→720p；ratio 空→16:9
- 状态：透传服务商 status（queued/running/succeeded/failed/expired）

### Google Veo 3.1（视频）— [google_video.go](../../backend/internal/provider/google_video.go)
- 注册：`RegisterVideoProvider("google", ...)`，模型 `veo-3.1-generate-preview`
- 端点：`https://generativelanguage.googleapis.com/v1beta/models/<model>:predictLongRunning`（创建，Header `x-goog-api-key`）/ `GET .../v1beta/<operationName>`（Long Running Operation 查询）
- 支持 referenceImages（≤3，`referenceType:"asset"`）—— Veo 独有
- 约束：ratio 仅 16:9/9:16（否则强制 16:9）；duration `5→6`、`7→8`、`>8→8`；1080p/4k 强制 duration=8
- **支持 HTTP_PROXY**，超时 300s

## 积分公式（精确，改造定价时直接改这里）

### 图像 [pricing.go](../../backend/internal/api/pricing.go) `ImagePricingConfig`
| 模型 | 0.5K | 1K | 2K | 4K |
|------|------|----|----|----|
| gemini-3-pro-image-preview (Nanobanana Pro) | — | 10 | 12 | 20 |
| gemini-3.1-flash-image-preview (Nanobanana 2) | 3 | 6 | 8 | 12 |
| doubao-seedream-4-5 (Seedream-4.5) | — | — | 6 | 10 |

未知 model/size：返回该模型档位最大值，再不行返回 10。

### 电商
`GetEcommerceCredits(size, count) = GetImageCredits(Seedream45, size) × count`

### 视频（由各 provider 的 `CalculateCredits` 实现）
- **Seedance**（volcengine_video.go）：`base{480p:6, 720p:10, 1080p:16}[resolution] × duration`，若 generateAudio 再 `×1.2`，最后 `int()`
- **Veo 3.1**（google_video.go，`VeoCreditsPerSecond`）：`{720p:45, 1080p:65, 4k:90}[resolution] × duration`（原生含音频，**不加价**）

**汇率**：`1 元 = 10 钻`（`exchange_rate: "1=10"`）。

## 模型 ID 速查

| 用途 | 模型 ID | 显示名 |
|------|---------|--------|
| 图像（默认优先） | `gemini-3-pro-image-preview` | 🍌 Nanobanana Pro |
| 图像（快） | `gemini-3.1-flash-image-preview` | 🍌 Nanobanana 2 |
| 图像（电商默认） | `doubao-seedream-4-5` | Seedream-4.5 |
| 视频 | `doubao-seedance-1-5-pro-251215` | Seedance-1.5 |
| 视频 | `veo-3.1-generate-preview` | Veo 3.1 |
| 反推提示词（内部） | `doubao-seed-2-0-pro-260215` | （豆包多模态，硬编码） |
| 提示词优化（内部） | `deepseek-chat`（env `DEEPSEEK_MODEL`） | — |

> 模型常量定义在 [pricing.go](../../backend/internal/api/pricing.go)（`ModelNanobanana` 等），便于全局引用。
