# 01 · 架构总览

## 一句话定位

小野AI 是一个**开源多模态 AI 内容创作平台**：图像/视频生成 + 电商组图 + 灵感社区 + 积分经济 + 用户体系。前端两套 Vue 3 SPA（用户端 / 管理后台），后端 Go/Gin 单体。

## 技术栈

| 层级 | 技术 | 版本/说明 |
|------|------|-----------|
| 前端（用户端） | Vue 3 + Pinia + Naive UI + Vue Router + Vite | **纯 JS（非 TS）**，vue-i18n 国际化 |
| 前端（管理后台） | Vue 3 + Ant Design Vue | 独立小 SPA，无 Pinia/无 vue-router |
| 后端 | Go 1.25 + Gin + GORM | 模块名 `google-ai-proxy`（历史遗留） |
| 数据库 | MySQL 8.0 | 自定义 SQL 迁移器，**非** AutoMigrate |
| 鉴权 | JWT（golang-jwt/v5）+ bcrypt | 用户端 JWT，管理后台静态 token |
| 存储 | 阿里云 OSS | 图/视频统一存 OSS，返回公开 URL |
| AI 供应商 | Google Gemini / 火山引擎（豆包 Seedream/Seedance / Veo） | 可插拔 provider 注册表 |
| 辅助 AI | DeepSeek（提示词优化）、豆包（反推提示词） | |
| 支付 | Linux.do Credit（EPay 协议） | 仅 Linux.do OAuth 用户可用 |
| 邮件 | SMTP（QQ/163/Gmail） | 验证码注册登录 |

## 目录结构

```
xiaoye-ai/
├── backend/                        # Go 后端
│   ├── main.go                     # 入口：env加载 + 初始化 + 路由 + worker
│   ├── go.mod                      # module: google-ai-proxy
│   ├── .env.example                # 环境变量模板
│   ├── cmd/                        # 独立 CLI 工具
│   │   ├── keygen/                 # 生成 License 兑换密钥
│   │   └── import_inspiration_full_json/  # 批量导入灵感帖
│   ├── internal/
│   │   ├── api/                    # HTTP handlers + 中间件 + 共享工具
│   │   │   ├── admin/              # 管理后台 API（审核）
│   │   │   ├── generate_handlers.go# 统一生成入口 ★
│   │   │   ├── pricing.go          # 积分定价 ★
│   │   │   ├── types.go            # 请求/响应 DTO ★
│   │   │   └── ...
│   │   ├── auth/                   # JWT + bcrypt + LicenseKey
│   │   ├── config/                 # 环境变量 getter
│   │   ├── db/                     # GORM 模型 + 迁移器 ★
│   │   ├── email/                  # SMTP
│   │   ├── payment/                # 支付 provider 抽象 + Linux.do 实现
│   │   ├── provider/               # AI 供应商适配 ★
│   │   └── storage/                # OSS
│   └── migrations/                 # 26 个 SQL 迁移脚本
├── frontend/                       # 用户端 Vue SPA
│   └── src/
│       ├── main.js                 # 入口（先拉 pricing 再挂载）
│       ├── router/index.js         # 路由 + SEO meta + 鉴权守卫
│       ├── layouts/AppLayout.vue   # 外壳布局
│       ├── views/                  # 页面
│       ├── components/             # 复用组件
│       ├── composables/            # useGenerate / useInspiration
│       ├── stores/                 # Pinia（8 个）
│       └── locales/                # zh.json / en.json
├── frontend-admin/                 # 管理后台 Vue SPA
│   └── src/
│       ├── main.js                 # 仅 Antd，无路由无 Pinia
│       └── components/AdminReviewPage.vue  # 审核台
├── docs/                           # 文档（本指南在此）
└── LICENSE                         # AGPL-3.0
```

## 启动流程（后端 [main.go](../../backend/main.go)）

```
1. 加载 .env（按顺序找：.env.${APP_ENV} → .env → backend/.env → /opt/nanobanana/.env → /etc/google-ai-proxy/.env）
2. auth.InitSecretKey()      # 读 JWT_SECRET，空则 log.Fatal（服务起不来）
3. db.InitDB()               # 连 MySQL + runMigrations()（建 schema_migrations + 跑未应用的 .sql）
4. email.InitEmail()         # 读 SMTP 配置
5. storage.InitOSS()         # 读 OSS 配置（失败仅 warning，不阻断）
6. 启动 3 个后台 worker:
     - StartVideoTaskPoller()      # 每 10s 轮询视频任务状态
     - StartVerificationCleanup()  # 每 1h 清过期验证码
     - StartGenerationCleanup()    # 每 1h 清 30 天前 failed 记录 + 3 天前 api_logs
     （oauth_handlers.go init() 另起一个每 5m 清 OAuth state）
7. gin.Default() + CORS（AllowOrigins 取自 CORS_ORIGINS）
8. 装配 /api 路由（公开 / 用户鉴权 / 管理员鉴权 三组）
9. r.Run(":" + PORT)         # 默认 :8092
```

## 请求/响应通用约定

- 所有 API 前缀 `/api`
- 鉴权头：用户端 `Authorization: Bearer <jwt>`；管理后台 `X-Admin-Token: <token>`
- 成功响应：直接返回业务 JSON（无统一信封）
- 失败响应：`{ "error": "<中文错误信息>" }`，HTTP 状态码语义化（401/402/403/400/500/503）
- 时间字段：DB 存 `datetime`，对外响应多为 `UnixMilli()`（见 `GenerationResponse`）

## 分层依赖关系

```
main.go
  └─ api (handlers)
       ├─ auth          (JWT/密码)
       ├─ db            (GORM 模型 + 事务)
       ├─ provider      (AI 供应商，注册表)
       ├─ storage       (OSS 上传/下载)
       ├─ email         (验证码邮件)
       └─ payment       (Linux.do 支付)
            └─ config    (env getter，被各层共用)
```

> 各层之间基本只通过 `db.DB`（全局 GORM 实例）和 provider 注册表交互，耦合度低。handler 层是业务编排层，几乎所有跨模块逻辑都在 api 包内。
