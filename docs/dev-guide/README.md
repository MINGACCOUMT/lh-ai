# 小野AI 开发指南（Development Guide）

本目录是**基于 xiaoye-ai 进行个性化改造的开发参考文档**，目标是让你（和 AI）在提需求/改代码时，能快速定位"改哪个文件、走哪条链路、有哪些坑"。

所有文档与**当前代码**对齐（撰写时已逐文件核实源码）。如代码后续演进，请同步更新本目录。

## 文档导航

| 文档 | 内容 | 什么时候看 |
|------|------|-----------|
| [01-overview.md](01-overview.md) | 架构总览、技术栈、目录结构、启动流程 | **先读这篇**，建立全局认知 |
| [02-backend.md](02-backend.md) | 后端分层、路由表、中间件、后台 worker、各 handler 职责、infra | 改后端逻辑、加接口、调业务流程 |
| [03-frontend.md](03-frontend.md) | 前端启动、路由布局、Pinia stores、composables、页面、组件、构建配置 | 改前端交互、加页面、接接口 |
| [04-data-model.md](04-data-model.md) | 数据库表与字段、迁移机制、26 个迁移脚本索引 | 改表结构、写新迁移、查字段含义 |
| [05-api-reference.md](05-api-reference.md) | 全部 HTTP 接口清单（方法/路径/鉴权/入参/出参） | 前后端联调、写客户端、对照接口 |
| [06-providers.md](06-providers.md) | AI 供应商抽象层、各 provider 实现、模型 ID、积分公式 | 接新模型、改生成逻辑、调价格 |
| [07-customization.md](07-customization.md) | **改造指引**：常见改造场景、扩展点、踩坑清单 | 动手改造前**必读** |

## 快速速查

- **后端入口**：[backend/main.go](../../backend/main.go)（路由装配 + worker 启动）
- **统一生成入口**：[backend/internal/api/generate_handlers.go](../../backend/internal/api/generate_handlers.go) → `POST /api/generate`
- **数据模型**：[backend/internal/db/db.go](../../backend/internal/db/db.go)
- **AI 供应商注册**：[backend/internal/provider/interface.go](../../backend/internal/provider/interface.go) + [video_interface.go](../../backend/internal/provider/video_interface.go)
- **积分定价**：[backend/internal/api/pricing.go](../../backend/internal/api/pricing.go)
- **前端路由**：[frontend/src/router/index.js](../../frontend/src/router/index.js)
- **前端生成页**：[frontend/src/views/Generate.vue](../../frontend/src/views/Generate.vue)
- **管理后台**：[frontend-admin/src/components/AdminReviewPage.vue](../../frontend-admin/src/components/AdminReviewPage.vue)

## 关键约定（先记住这几条）

1. **Go module 名是历史的 `google-ai-proxy`**，不是 xiaoye-ai。所有 import 路径以 `google-ai-proxy/internal/...` 开头——改名需全局替换，谨慎。
2. **数据库不用 GORM AutoMigrate**，而是自定义 SQL 文件运行器（[db.go 的 runMigrations](../../backend/internal/db/db.go)）。改表结构 = 新增 `migrations/NNN_xxx.sql`。
3. **积分（credits/钻石）是核心经济模型**，所有扣费走"原子条件更新 + 流水 + 失败退款"三件套，改造涉及扣费务必保持闭环。
4. **两套独立鉴权**：用户端 JWT（`Authorization: Bearer`），管理后台 `X-Admin-Token`。
5. **环境变量零散**：配置全靠 `os.Getenv`（[config.go](../../backend/internal/config/config.go)），新增配置 = 加 env + 加 getter。

> 文档撰写语言为中文，与代码注释风格一致。
