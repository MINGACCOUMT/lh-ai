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
| ④ 资产库 | ⏳ 待做 | 人物/场景图生成（小说级复用，喂给视频） |
| 视频 B-3 | ⏳ 待做 | shots/资产 → 视频模型 → 拼接 |

**数据表**：`novels`、`novel_chapters`(+outline/characters/scenes/analysis_status)、`novel_plots`(+storyboard_status)、`novel_shots`(plot_id)。

**关键 bug 修复记录**（过程踩坑）：切章正则空行吞标题、dev_code 死代码、keygen 未加载 JWT_SECRET、OSS 阻止公共访问、chat 静默失败、novel_plots 无 updated_at、shots 唯一键 chapter_id→plot_id、解析 characters/scenes 数组类型、**GBK/GB18030 自动转码**（大小说非 UTF-8 上传）、**空正文章节过滤**（卷首"第X卷"标题误切为空章节）。

> 注：上传限制 50MB；非 UTF-8(.txt) 自动按 GBK/GB18030 解码；空正文章节（卷首/分节标题）自动滤除并重排序号。

## 四、待办（按优先级）

1. **④ 资产库**：novel_characters/novel_scenes 表 + 生图接口（角色/场景图，小说级复用）+ 分镜页"生成人物/场景图"按钮。
2. **视频 B-3**：图→视频模型→拼接（需先解决 Veo 通道或用 sora-2/Seedance）。
3. 章节切分：楔子/序章保留（首标记前内容当序章）、正文"第X章"误切启发式。
4. （可选）整理 dev-guide 文档同步新模块。

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
