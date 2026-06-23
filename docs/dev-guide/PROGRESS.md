# 小野AI 个性化改造 · 当前进度

> 仓库：`github.com/MINGACCOUMT/lh-ai`（私有 fork），分支 `feat/novel-storyboard`。  
> 上游：`github.com/capybara-zy/xiaoye-ai`（公共，作为 `upstream`，勿推）。  
> 更新日期：2026-06-21。

## 一、开发环境（已跑通）

| 项 | 配置 |
|----|------|
| MySQL | `192.168.5.26:33906/xiaoye_ai`（root，utf8mb4），31 个迁移全 applied |
| OSS | 桶 `lc-veo`（oss-cn-shanghai，**已设公共读**），图片上传+公开访问 OK |
| 后端 | `:8092`，`go run -C backend .` |
| 前端 | `:5173`（vite，/api 代理 :8092） |
| 测试号 | `video@test.local` / `Test1234!`（1000+ 钻） |
| JWT/OSS/DB 等 | 见 `backend/.env`（gitignored） |

## 二、新增的 AI 供应商接入（中转站/直连）

| 文件 | 作用 | 配置 |
|------|------|------|
| `provider/relay_openai.go` | Gemini 图像走 ai-tudou（OpenAI /v1/chat/completions） | `OPENAI_BASE_URL`/`OPENAI_API_KEY` |
| `provider/gpt_image.go` | gpt-image-2 走 koramkoin（/v1/images/generations） | `GPT_IMAGE_BASE_URL`/`GPT_IMAGE_API_KEY` |
| `provider/relay_video.go` | Veo/Seedance/sora-2 视频（ai-tudou /v1/videos） | `OPENAI_*` |
| `novel/chat.go` | **小说 LLM：智谱 glm-5.2**（Anthropic /v1/messages） | `NOVEL_LLM_BASE_URL=https://open.bigmodel.cn/api/anthropic`、`NOVEL_LLM_API_KEY`、`NOVEL_LLM_MODEL=glm-5.2` |

**图片生成**：gpt-image-2（koramkoin）✅ 端到端验证通过（生图→OSS→公开URL）。
**视频生成**：provider 代码就绪；**Veo 卡在 ai-tudou 的 svip 组无 veo 通道**（需中转站开通），sora-2 可用。

## 三、小说→分镜 功能（v2 设计）

设计文档：[specs/2026-06-21-novel-storyboard-design-v2.md](../superpowers/specs/2026-06-21-novel-storyboard-design-v2.md)

**结构**：小说 → 章 → **情节(plot)** → 分镜(shot)；章级有 大纲/人物画像/场景。

| 阶段 | 状态 | 说明 |
|------|------|------|
| ① 解析 analyze | ✅ 完成+验证 | glm-5.2 产出 大纲+人物画像+场景+情节列表（人物/场景数组已归一化） |
| ② 情节下分镜 | ✅ 完成+验证 | `/plots/:id/storyboard`，每情节自适应 2~5 镜 |
| ③ 前端章节详情 | ✅ 完成 | 解析按钮→大纲/人物/场景→情节列表→情节下分镜+镜头编辑 |
| ④ 资产库 | ✅ 完成 | 资产存 generations 表（novel_id 区分小说库/公共池）；分镜页生成人物/场景图；Assets 页加小说名/公共池筛选 |
| Veo 3.1 视频 | ✅ API 适配 | relay_video.go 匹配新 API（/v1/videos/generations + /v1/tasks + veo3.1-720p/1080p）；卡中转站通道 |
| 视频 B-3 | ⏳ 待做 | shots/资产 → 视频模型 → 拼接 |

**数据表**：`novels`、`novel_chapters`(+outline/characters/scenes/analysis_status)、`novel_plots`(+storyboard_status)、`novel_shots`(plot_id)。

**关键 bug 修复记录**（过程踩坑）：切章正则空行吞标题、dev_code 死代码、keygen 未加载 JWT_SECRET、OSS 阻止公共访问、chat 静默失败、novel_plots 无 updated_at、shots 唯一键 chapter_id→plot_id、解析 characters/scenes 数组类型、**GBK/GB18030 自动转码**（大小说非 UTF-8 上传）、**空正文章节过滤**（卷首"第X卷"标题误切为空章节）、**useMessage() 替代 window.$message**（toast 从不显示）、**章节状态 analysis_status vs storyboard_status 混淆**、**切换小说/章节缓存旧数据**（route watch + store reset）、**查询性能优化 Select**（排除 raw_content/content 大文本，10ms 级响应）、**goroutine panic 恢复 + 启动清理**（卡住状态兜底）、**gpt-image-2 超时 300s→180s**（单张 >3 分钟判失败）。

> 注：上传限制 50MB；非 UTF-8(.txt) 自动按 GBK/GB18030 解码；空正文章节自动滤除；切换页面有 toast 反馈（解析/分镜/资产 开始+完成）；AssetPicker 从素材库选参考图。

## 四、待办（按优先级）

1. **视频 B-3**：图→视频模型→拼接（Veo 3.1 新 API 通道已通，上游偶尔过载；ffmpeg 已安装）。
2. 章节切分：楔子/序章保留（首标记前内容当序章）、正文"第X章"误切启发式。
3. （可选）整理 dev-guide 文档同步新模块。

## 五、今日新增功能（2026-06-23）

- **提示词灵感库**：`GET /api/prompts`（分页+搜索 prompts-all.json）；前端 `/prompts` 页面（瀑布流+搜索+复制提示词+分享到广场）；分镜编辑页"💡灵感"按钮（从灵感库选提示词复制到剪贴板）。
- **AssetPicker 增强**：参考图/首帧/尾帧均可从素材库选（弹出选择：上传 vs 素材库）；修复 images 字段解析（API 返回数组不再 JSON.parse 报错）。
- **gpt-image-2 参考图修复**：有参考图时调 `/v1/images/edits`（之前忽略 InputImages，只调 `/v1/images/generations`），参考图传 OSS URL。
- **章节列表状态修复**：徽章用 `analysis_status`（未解析/解析中/已解析/失败），不再用 `storyboard_status`；chLite 返回 analysis_status + assets_status。
- **切换缓存修复**：store openNovel/openChapter 开头清空旧数据 + route watch 监听参数变化重载。
- **toast 反馈**：useMessage() 替代 window.$message（之前全失效）；解析/分镜/资产 开始+完成都有提示。
- **查询性能优化**：Select 排除 raw_content/content（10ms 级响应，之前因拉大文本 blob 慢）。
- **兜底机制**：4 个 goroutine 加 defer recover；后端启动清理卡住状态（analyzing/generating/extracting → failed）。
- **gpt-image-2 超时** 300s→180s（单张 >3 分钟判失败）。
- **Veo 3.1 新 API 适配**：relay_video.go 匹配 `/v1/videos/generations` + `/v1/tasks` + `veo3.1-720p/1080p`。

## 五、运行/测试速查

```bash
# 后端
cd backend && go run .                      # :8092
# 前端
cd frontend && npm run dev                  # :5173
# 测解析（API）
curl -X POST http://localhost:8092/api/novel/chapters/<id>/analyze -H "Authorization: Bearer <token>"
# 测情节分镜
curl -X POST http://localhost:8092/api/novel/plots/<id>/storyboard -H "Authorization: Bearer <token>"
```

> 所有进度已推送 `lh-ai` 的 `feat/novel-storyboard` 分支，丢不了。
