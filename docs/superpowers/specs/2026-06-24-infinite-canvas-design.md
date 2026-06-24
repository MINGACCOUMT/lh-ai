# 无限画布（Infinite Canvas）设计

> 日期：2026-06-24 · 方案 A（Vue Flow）

## 1. 目标

在 xiaoye-ai 中新增一个 **AI 创作工作台**——无限画布。用户拖拽图片/视频/文字/提示词节点到画布上，连线触发 AI 生成，结果作为新节点出现。画布布局后端持久化，支持多项目。

## 2. 技术选型

- 前端画布引擎：**@vue-flow/core** + `@vue-flow/background` + `@vue-flow/controls` + `@vue-flow/minimap`
- 节点：Vue 3 自定义组件（4 种节点类型）
- 后端：Go/Gin/GORM/MySQL（复用现有架构）
- 存储：复用现有 OSS（图片/视频结果）
- 生成：复用现有 `/api/generate`（image/video）

## 3. 节点类型

| 节点 | 功能 | 数据 |
|------|------|------|
| **图片节点** | 显示一张图片（OSS URL 或上传）；可拖出连线作为参考图 | `{imageUrl, thumbnailUrl}` |
| **视频节点** | 显示视频缩略图 + 可播放 | `{videoUrl, thumbnailUrl}` |
| **文字节点** | 纯文字标注/分组说明 | `{text}` |
| **提示词节点** | 输入提示词 + 选择模型 + 生成按钮；接收上游图片作为参考图 | `{prompt, model, params, resultUrl}` |

## 4. 连线/拖拽生成

- 用户从**图片节点**拖一条连线到**提示词节点**的"参考图"输入端口 → 提示词节点自动获取参考图 URL
- 提示词节点点"生成" → 调 `/api/generate`（type=image, images=[参考图URL], model, prompt）→ 轮询 → 结果作为新**图片节点**出现在提示词节点下方，自动连线
- 支持多参考图：多条图片连线汇入同一提示词节点

## 5. 数据模型

### `canvas_projects`
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint64 PK | |
| user_id | bigint index | 所属用户 |
| name | varchar(200) | 项目名 |
| status | varchar(20) | active/archived |
| created_at / updated_at | datetime | |

### `canvas_data`（单个 JSON 字段存整个画布布局）
画布布局序列化为一个 JSON 存在 `canvas_projects.layout` TEXT 字段：
```json
{
  "nodes": [
    {"id":"1","type":"image","position":{"x":100,"y":200},"data":{"imageUrl":"https://..."}},
    {"id":"2","type":"prompt","position":{"x":400,"y":200},"data":{"prompt":"a cat","model":"gpt-image-2"}}
  ],
  "edges": [
    {"id":"e1","source":"1","target":"2","sourceHandle":null,"targetHandle":"ref"}
  ]
}
```
> 用 JSON 存布局而非拆表——画布操作频繁（拖拽位置），JSON 一次保存比逐节点更新高效。

## 6. API（/api/canvas，JWT）

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/canvas` | 项目列表 |
| POST | `/canvas` | 新建项目 |
| GET | `/canvas/:id` | 获取项目（含 layout JSON） |
| PUT | `/canvas/:id` | 保存（更新 layout + name） |
| DELETE | `/canvas/:id` | 删除项目 |

生成不单独开接口——前端直接调现有 `/api/generate`，结果在前端作为新节点添加，保存时随 layout 一起持久化。

## 7. 前端结构

```
frontend/src/
├── views/Canvas.vue              # 画布主页面（Vue Flow + 工具栏 + 侧边项目列表）
├── components/canvas/
│   ├── ImageNode.vue             # 图片节点
│   ├── VideoNode.vue             # 视频节点
│   ├── TextNode.vue              # 文字节点
│   ├── PromptNode.vue            # 提示词+生成节点
│   ├── CanvasToolbar.vue         # 顶部工具栏（添加节点/缩放/撤销/保存）
│   └── CanvasSidebar.vue         # 左侧项目列表
├── stores/canvas.js              # Pinia store（项目列表/当前项目/layout）
└── composables/useCanvas.js      # 画布操作（添加节点/生成/保存）
```

路由：`/canvas` + `/canvas/:projectId`

## 8. 开发计划（分 3 期）

### Phase 1：画布骨架 + 图片/文字节点（最小可用）
- 装 @vue-flow/* 依赖
- Canvas.vue 页面 + 路由 + 导航
- 图片节点（拖入/粘贴 URL → 显示）+ 文字节点
- 拖拽/缩放/选中/删除基本操作
- 后端：canvas_projects 表 + CRUD API
- 前端：项目列表 + 新建/打开/保存

### Phase 2：提示词节点 + 连线生成
- 提示词节点（输入提示词 + 模型选择 + 生成按钮）
- 连线系统（图片 → 提示词 = 参考图）
- 生成流程：调 /api/generate → 轮询 → 结果作为图片节点出现
- 自动布局（结果节点出现在提示词下方）

### Phase 3：视频节点 + 打磨
- 视频节点（从素材库/生成结果添加 → 预览）
- 撤销/重做（Vue Flow 自带）
- 小地图 + 导出图片
- 右键菜单（删除/复制/设为参考图）

## 9. 不在本版范围

- 节点裁剪/蒙版/分割（infinite-canvas 的高级编辑功能）
- 工作流模板
- 协作/共享画布
- 节点角度/堆叠
