# 小说→分镜→资产 设计 v2（演进版）

> 日期：2026-06-21 · 取代 v1（`...-design.md`）。  
> 本版相对 v1 的关键变化：① 加「情节(plot)」层；② 解析/分镜拆成两阶段；③ 引入小说级资产库（人物图/场景图），分镜可触发生成；④ LLM 换智谱 glm-5.2（Anthropic 端点）。

## 1. 目标（本版范围）

小说 .txt → 切章 → **解析**（大纲+人物画像+场景+**情节列表**）→ **生成分镜**（情节下、自适应数量）→ **生成资产**（人物图/场景图，小说级复用）。
**视频生成（B-3）不在本版**，单开子项目。

## 2. 结构层级

```
小说 (novel)
 └ 章 (chapter)
     ├ 解析产出（章级）：大纲 outline / 人物画像 characters / 场景 scenes
     └ 情节 (plot)        ← 解析时 LLM 切出，一章 N 个（标题+简述）
         └ 分镜 (shot)    ← 生成分镜时，每个情节下 LLM 自适应产出若干镜头
资产库（小说级，跨章复用）
 ├ 角色资产 novel_characters  （name + description + image_url）
 └ 场景资产 novel_scenes      （name + description + image_url）
```

## 3. 阶段（手动逐步）

| 阶段 | 触发 | LLM 产出 | 存储 |
|------|------|---------|------|
| ① 解析 analyze | 章节详情点"解析" | 大纲 + 人物画像 + 场景 + **情节列表** | chapter.outline/characters/scenes + novel_plots |
| ② 生成分镜 storyboard | 解析后点"生成分镜" | 每个情节下的镜头（数量 LLM 按情节长短自适应） | novel_shots（挂 plot_id） |
| ③ 生成资产 assets | 分镜页点"生成人物/场景图" | 每个角色/场景的图像（图像模型） | novel_characters / novel_scenes（小说级，复用） |

> 人物/场景资产是**小说级**：第1章生成的"李云起"图，第2章复用同一张 → 全书人物一致。

## 4. 数据模型

- `novels`（已建）：id, user_id, title, source_filename, raw_content, chapter_count, status。
- `novel_chapters`（已建 + 已加 outline；本版再加）：`characters` text, `scenes` text, `analysis_status` varchar(none/analyzing/ready/failed)。
- `novel_plots`（**新**）：id, chapter_id(index), plot_index, title, summary, created_at。unique(chapter_id, plot_index)。
- `novel_shots`（**改**：chapter_id → `plot_id`）：id, plot_id(index), shot_index, scene, characters, prompt, dialogue, camera, image_url, video_url, created_at。unique(plot_id, shot_index)。
- `novel_characters`（**新**，资产）：id, novel_id(index), name, description, image_url, created_at。unique(novel_id, name)。
- `novel_scenes`（**新**，资产）：id, novel_id(index), name, description, image_url, created_at。unique(novel_id, name)。

## 5. API（/api/novel，JWT）

| 方法 | 路径 | 作用 |
|------|------|------|
| POST | `/upload` | 上传 .txt 切章（已建） |
| GET | `/:id?limit=&offset=` | 小说+章节分页（已建） |
| DELETE | `/:id` | 删除（已建） |
| POST | `/chapters/:id/analyze` | **解析**：大纲+人物+场景+情节（新） |
| GET | `/chapters/:id` | 章节详情：analysis + plots + shots（新） |
| POST | `/plots/:id/storyboard` | **生成分镜**：该情节下自适应镜头（新） |
| POST | `/chapters/:id/assets` | 生成本章人物/场景图→资产库（新，资产生成子项） |
| PUT | `/shots/:id` | 编辑分镜（已建） |

## 6. LLM

- 解析/分镜：**智谱 glm-5.2**，Anthropic 兼容端点 `https://open.bigmodel.cn/api/anthropic`（`/v1/messages`, `x-api-key`）。已通。
- 资产图：图像模型（gpt-image-2 / Gemini，已有 provider）。

## 7. 已完成 vs 待做

**已完成**：novels/chapters/shots 表+模型、上传切章（正则+空行修复）、CRUD、章节分页、glm-5.2 接入、outline 字段、UI 三页+分页+大纲展示+解析按钮、50MB 上限。

**待做（本版）**：
1. 加 `novel_plots` 表 + shots 改挂 plot_id（迁移 029/030）。
2. 解析改成产 **情节列表**（+大纲/人物/场景），存 novel_chapters + novel_plots。拆 `/analyze` 接口。
3. 生成分镜改成**情节下**自适应（`/plots/:id/storyboard`）。
4. 资产库：novel_characters/novel_scenes + 生图接口 + 分镜页"生成人物/场景图"按钮。
5. 前端：章节详情展示情节列表；情节下展开分镜；资产生成 UI。

## 8. 后续子项目（不在本版）

- B-3 视频：shots/资产 → 视频模型 → 拼接（九宫格概念转为"本章全部镜头拼视频"）。
- 道具资产库（novel_props）。
