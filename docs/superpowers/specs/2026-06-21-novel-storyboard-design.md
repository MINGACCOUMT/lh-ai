# 小说 → 分镜（流水线前半段）设计

> 日期：2026-06-21  
> 子项目：B-1（流水线前半段）  
> 上游大方向：小说 → 视频（上传 → 切章 → 分镜 → 生图 → 生视频 → 拼接）。本 spec 仅覆盖**前半段**。

## 1. 背景与目标

用户上传小说 .txt，系统自动切章，并为每章用 LLM 抽取 **9 个分镜**（场景 / 人物 / 图片提示词 / 对白旁白 / 镜头运动），用户可在前端逐字段编辑。本子项目是整条流水线的**地基**，为后续「分镜生图」「图生视频+拼接」提供结构化数据，自身可独立交付、独立验收。

## 2. 范围

**包含（In Scope）**
- 上传 .txt 小说（文件上传 + 解析）
- 正则自动切章
- 小说 / 章节的增删查
- 每章按需触发 LLM 抽取 9 个分镜（异步）
- 分镜的查看与逐字段编辑、保存
- 相关积分扣费（失败退款）

**不包含（Out of Scope，后续子项目）**
- 分镜 → 图片生成（B-2）
- 图片 → 视频 + ffmpeg 拼接（B-3）
- 完整资源库：角色 / 场景 / 道具的**参考图**管理（A）
- 分镜 `characters` 本 spec 只存人物名（文本），不带参考图

## 3. 用户流程

```
[小说列表页] 上传 .txt
      │  POST /novel/upload（同步切章，建 novel + chapters）
      ▼
[小说列表页] 看到新小说 + 章节数
      │  选一本
      ▼
[章节列表页] 每章带状态徽章：未抽取 / 抽取中 / 就绪 / 失败
      │  点某章「生成分镜」
      ▼
POST /novel/chapter/:id/storyboard（异步任务启动，扣费）
      │  前端轮询 GET .../storyboard
      ▼
[分镜编辑页] 9 张分镜卡片（场景/人物/提示词/对白/镜头），可改 → PUT /novel/shot/:id
```

## 4. 架构

- **编排全在后端**，前端只触发 + 轮询 + 编辑。与现有图像/视频异步任务模式一致。
- **LLM**：走中转站 chat（`OPENAI_BASE_URL` + `OPENAI_API_KEY`，即 ai-tudou），模型名由 env `NOVEL_LLM_MODEL` 指定（默认 `deepseek-v4-flash`）。`response_format=json_object` 保证结构化。
  - 新增通用 helper `relayChat(model, systemPrompt, userContent) (string, error)`，供 novel 模块及后续复用。
- **文件存储**：.txt 正文存 DB（`novels.raw_content` longtext），不走 OSS（纯文本，DB 更简、便于重切/编辑）。
- **切章**：上传时 Go 正则同步切分，写入 `novel_chapters`。

## 5. 数据模型（3 张新表）

迁移文件：`backend/migrations/027_add_novel_system.sql`

### `novels`
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint64 PK | |
| user_id | bigint index | 所属用户 |
| title | varchar(200) | 小说标题（默认取文件名去扩展名） |
| source_filename | varchar(255) | 原始 .txt 文件名 |
| raw_content | longtext | 全文（切章依据、可重切） |
| chapter_count | int | 章节数 |
| status | varchar(20) default 'active' | active/archived |
| created_at / updated_at | datetime | |

### `novel_chapters`
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint64 PK | |
| novel_id | bigint index | |
| chapter_index | int | 顺序（从 1） |
| title | varchar(200) | "第X章 …" |
| content | longtext | 本章正文 |
| storyboard_status | varchar(20) default 'none' | none/extracting/ready/failed |
| created_at / updated_at | datetime | |
| 索引 | unique(novel_id, chapter_index) | |

### `novel_shots`
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint64 PK | |
| chapter_id | bigint index | |
| shot_index | int (1~9) | 分镜序号 |
| scene | varchar(1000) | 场景描述 |
| characters | varchar(500) | 人物（逗号分隔文本） |
| prompt | varchar(2000) | 图片提示词（分镜词）——后续生图核心输入 |
| dialogue | varchar(1000) | 对白/旁白 |
| camera | varchar(100) | 镜头运动（如 推进/平移/特写） |
| image_url | varchar(500) null | 预留：B-2 生图后填 |
| video_url | varchar(500) null | 预留：B-3 生视频后填 |
| created_at / updated_at | datetime | |
| 索引 | unique(chapter_id, shot_index) | |

> `image_url`/`video_url` 预先预留，B-2/B-3 直接 UPDATE，不改表。

## 6. API（前缀 `/api/novel`，全部 JWT）

| 方法 | 路径 | 入参 | 出参 |
|------|------|------|------|
| POST | `/novel/upload` | multipart `file`（.txt） | `{novel_id, title, chapter_count}` |
| GET | `/novel` | query: limit/offset | `{items:[{id,title,chapter_count,created_at}], total}` |
| GET | `/novel/:id` | — | novel + `chapters:[{id,index,title,storyboard_status}]` |
| DELETE | `/novel/:id` | — | 级联删 chapters + shots |
| POST | `/novel/chapter/:id/storyboard` | — | 触发异步抽取；`{status:"extracting"}`（扣费） |
| GET | `/novel/chapter/:id/storyboard` | — | `{status, shots:[{id,shot_index,scene,characters,prompt,dialogue,camera, image_url, video_url}]}` |
| PUT | `/novel/shot/:id` | 任意字段子集 | 更新后该 shot |

> 所有权校验：用户只能访问自己的 novel/chapter/shot（与 generations 一致）。

### 抽取响应契约（LLM 输出）
`relayChat` 以 `response_format=json_object` 要求模型返回：
```json
{"shots":[
  {"scene":"...","characters":"A, B","prompt":"...","dialogue":"...","camera":"推进"},
  ... // 共 9 个
]}
```
解析为 9 行写入 `novel_shots`（不足/超出 9 时见 §8）。

## 7. 关键决策

| 决策 | 取值 | 备注 |
|------|------|------|
| 文件存储 | DB longtext | 纯文本不走 OSS |
| 切章 | 上传时正则同步 | 快；无匹配见 §8 |
| 抽取 | 异步任务 | LLM ~15s，不阻塞 |
| LLM | 中转站 chat，`NOVEL_LLM_MODEL` 默认 `deepseek-v4-flash` | DeepSeek 未配 key，复用中转站 |
| 积分 | `NOVEL_STORYBOARD_CREDITS` 默认 **1** | 触发时扣，失败退款（同 generate） |
| 章节正文上限 | `NOVEL_CHAPTER_MAX_CHARS` 默认 **12000** | 超长截断送 LLM，避免上下文爆 |

## 8. 错误处理

- **上传**：仅 .txt；大小上限（如 10MB）；编码强制 utf-8（非 utf-8 报错提示）。
- **无章节标题匹配**：正则未命中任何「第X章 / 第X回 / Chapter X」→ 整文作为 **1 章**，`title` 取「全文」，并在响应里 `warning: "未识别到章节标题，已按整文处理"`。
- **抽取失败**：LLM 调用失败 / JSON 解析失败 / 返回非 9 个 → `storyboard_status=failed` + 退款；用户可重试。重试会**覆盖**该章旧分镜（编辑丢失，前端二次确认）。
- **LLM 返回不足/超 9**：取前 9 个；不足 9 则存实际数量（前端如实展示，B-2 生图时按实际 shot 数处理）。
- **并发**：同一章正在 extracting 时，再次触发返回 409「抽取中」。

## 9. 前端

- 新增导航项「小说」。
- `NovelList.vue`：小说列表 + 上传入口（拖拽 .txt）。
- `NovelDetail.vue`：章节列表，每章状态徽章 + 「生成分镜」按钮 + 进入编辑。
- `ChapterStoryboard.vue`：9 张分镜卡片（可编辑 scene/characters/prompt/dialogue/camera），保存按钮；抽取中显示 loading + 轮询。
- 复用：Naive UI、Pinia、i18n、axios token 注入（沿用 stores 模式）。

## 10. 配置项（新增 env，加到 config.go + .env.example）

| 变量 | 默认 | 说明 |
|------|------|------|
| `NOVEL_LLM_MODEL` | `deepseek-v4-flash` | 抽分镜用的中转站 chat 模型 |
| `NOVEL_STORYBOARD_CREDITS` | `1` | 每章抽取扣钻 |
| `NOVEL_CHAPTER_MAX_CHARS` | `12000` | 送 LLM 的章节正文上限 |

## 11. 验收标准

1. 上传一个含「第X章」的 .txt → 正确切出多章，章节列表可见。
2. 上传一个无标题的 .txt → 整文 1 章 + warning。
3. 某章点「生成分镜」→ 扣 1 钻 → 异步产出 9 个分镜（5 字段齐全）→ 前端展示。
4. 编辑任一分镜字段 → 保存 → 重新打开仍在。
5. 抽取失败（断网/坏 key）→ 状态 failed + 钻石退回 + 可重试。
6. 越权访问他人小说 → 403/404。

## 12. 后续子项目（依赖本 spec）

- **A 资源库**：角色/场景/道具参考图 CRUD；分镜 `characters` 升级为关联角色库。
- **B-2 分镜生图**：按 shot_index 逐个生图，写回 `image_url`；接入资源库参考图保一致性。
- **B-3 图生视频 + 拼接**：每图调视频模型生片段（带 `camera` 运镜），ffmpeg 拼成章节视频。
