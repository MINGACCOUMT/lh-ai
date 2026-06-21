# 07 · 个性化改造指引

> 动手改造前**必读**。本篇给常见改造场景的具体落点 + 已知踩坑清单。先看踩坑，避免踩雷。

## ⚠️ 踩坑清单（先看这个）

1. **`INSPIRATION_AUTO_APPROVE` 默认值矛盾** — [.env.example](../../backend/.env.example) 写的是 `false`，但 [inspiration_handlers.go](../../backend/internal/api/inspiration_handlers.go) 的 `isInspirationAutoApprove()` 在**变量为空时默认返回 `true`**（自动审核通过）。所以：没设这个变量 = 自动通过；想强制人工审核必须**显式设 `INSPIRATION_AUTO_APPROVE=false`**。
2. **HTTP_PROXY 只对 Google 系生效** — 只有 [gemini.go](../../backend/internal/provider/gemini.go) 和 [google_video.go](../../backend/internal/provider/google_video.go) 实现了代理；火山引擎两个文件（[volcengine.go](../../backend/internal/provider/volcengine.go)、[volcengine_video.go](../../backend/internal/provider/volcengine_video.go)）**不走代理**。国内调火山没问题，但若整体走代理网关要注意火山直连。
3. **Go module 名是 `google-ai-proxy`** — 所有 import 路径 `google-ai-proxy/internal/...`。改名要全局替换，**且 go.mod 的 module 行也要改**，风险高，非必要不改。
4. **数据库不用 AutoMigrate** — 别以为加了 GORM struct tag 就自动建表/改列。改表 = 新增 [migrations/NNN_xxx.sql](../../backend/migrations/)。`splitSQL` 按 `;` 简单分割，**不支持存储过程和字符串内分号**。
5. **套餐金额在前后端各有一份，可能脱节** — 后端基准在 [payment_handlers.go](../../backend/internal/api/payment_handlers.go) `paymentPlans`：`starter ¥129/100钻`、`popular ¥559/500钻`、`pro ¥999/1000钻`。前端 [PricingModal.vue](../../frontend/src/components/PricingModal.vue) 自己写了一份展示。**改价务必两处同步**，否则用户看到的和实际扣的对不上。
6. **注册默认邀请码硬编码 `'VIP666DB'`** — 在 [AuthModal.vue](../../frontend/src/components/AuthModal.vue)。若做邀请体系改造注意。
7. **支付验签非恒定时间比较** — [linuxdo_credit.go](../../backend/internal/payment/linuxdo_credit.go) `VerifyNotify` 用普通 `==`（admin 那边用了 `subtle.ConstantTimeCompare`，这里没有）。理论上有时序攻击风险，关键业务建议改成恒定比较。
8. **QQ 邮箱 `"short response"` 被当成功** — [email.go](../../backend/internal/email/email.go) 的已知 workaround（Go issue #24845）。换邮件服务商时别误删。
9. **前端无全局 axios 拦截器** — 401 处理只在 `user.fetchUserInfo()`。新增需要鉴权的 API 调用时，要么走已有的 store/composable（自带 token 注入），要么手动拼 `Authorization` 头。
10. **两套独立 token 体系** — 用户端 `localStorage.token`（JWT），管理后台 `localStorage.admin_token`（X-Admin-Token），互不通用。

## 常见改造场景

### A. 接入一个新的图像模型
1. 在 [provider/](../../backend/internal/provider/) 新建 `xxx.go`，实现 `ImageGenerator`（多图则实现 `MultiImageGenerator`）
2. `init()` 里 `Register("<model-id>", &YourModel{...})`
3. 模型ID 常量加到 [pricing.go](../../backend/internal/api/pricing.go)，并加进 `ImagePricingConfig` 定价表 + `modelOrder`（控制列表排序）
4. 如需设为默认，改 `GetDefault()` 优先级数组
5. 前端 [ComposerBar.vue](../../frontend/src/components/ComposerBar.vue) 如有模型特有参数（如极端比例）需适配
6. 测试：启动后 `GET /api/models` 应出现新模型

### B. 接入一个新的视频模型
1. 新建 provider 文件实现 `VideoProvider`（5 个方法 + `CalculateCredits`）
2. `init()` 里 `RegisterVideoProvider("<providerName>", ...)` **且** 在 [video_interface.go](../../backend/internal/provider/video_interface.go) 的 `modelProviderMap` 加 `"<model-id>": "<providerName>"`
3. 定价：`CalculateCredits` 在 provider 内实现（参考 Seedance/Veo）；如要暴露给前端，在 [pricing.go](../../backend/internal/api/pricing.go) `GetPricing` 的 video 块加 `*_base_per_second`
4. 前端 ComposerBar 视频模型选项 + [generate_handlers.go](../../backend/internal/api/generate_handlers.go) `handleUnifiedVideoGenerate` 的默认模型如需改

### C. 调整积分定价
- 图像/电商：改 [pricing.go](../../backend/internal/api/pricing.go) `ImagePricingConfig` + 前端 PricingModal
- 视频：改对应 provider 的 `CalculateCredits`（公式在 [06-providers.md](06-providers.md)）
- 汇率：改 `GetPricing` 返回的 `exchange_rate`（业务语义，1元=10钻）
- 各功能固定扣费：提示词优化 `PROMPT_OPTIMIZE_CREDITS`(env)、反推 `reversePromptCredits=2`（[reverse_prompt_handlers.go](../../backend/internal/api/reverse_prompt_handlers.go)）、签到 `calcCheckinReward`、邀请 `inviteReward=10`/上限 `maxInviteCredits=500`、审核奖励 `getReviewRewardByPostType`(video+4/其他+2)

### D. 新增一个 API 接口
1. handler 写在 [internal/api/](../../backend/internal/api/)（或 `admin/`），DTO 加到 [types.go](../../backend/internal/api/types.go)
2. 在 [main.go](../../backend/main.go) 路由组注册（注意放公开 / 用户鉴权 / 管理员鉴权哪个组）
3. 涉及扣费 → 走 [utils.go](../../backend/internal/api/utils.go) 的 `recordCreditTransaction` + `refundCredits` 闭环（见 [02-backend.md](02-backend.md) §6）
4. 前端：加到对应 store（如 `user.js`）或 composable（`useGenerate.js`/`useInspiration.js`），用 `authHeaders()` 注入 token

### E. 改数据库表结构
1. 新增 [migrations/027_xxx.sql](../../backend/migrations/)（编号续接）
2. 在 [db.go](../../backend/internal/db/db.go) 加/改对应 GORM struct（保持字段 tag 与 SQL 一致，用于查询映射）
3. 重启服务自动应用（`schema_migrations` 记录防重复）
4. **别用** `db.AutoMigrate`（项目没有这个习惯）

### F. 改前端页面/交互
- 路由：[router/index.js](../../frontend/src/router/index.js)（加路由 + SEO meta key）
- 导航：[AppSidebar.vue](../../frontend/src/components/AppSidebar.vue) `navItems`
- 文案：[locales/zh.json](../../frontend/src/locales/) / [en.json](../../frontend/src/locales/)
- 状态：对应 Pinia store；跨页传参用 [composerDraft.js](../../frontend/src/stores/composerDraft.js)（参考灵感→生成的 remix 流程）
- 主题：[theme.js](../../frontend/src/stores/theme.js)（`<html data-theme>`）
- 生成主流程改 [Generate.vue](../../frontend/src/views/Generate.vue) + [ComposerBar.vue](../../frontend/src/components/ComposerBar.vue) + [useGenerate.js](../../frontend/src/composables/useGenerate.js)

### G. 改鉴权
- 用户 JWT：[auth.go](../../backend/internal/auth/auth.go)（有效期 7 天，`GenerateUserToken`/`ValidateUserToken`）+ [middleware.go](../../backend/internal/api/middleware.go)
- 管理后台：`ADMIN_TOKEN` env + [admin/middleware.go](../../backend/internal/api/admin/middleware.go)
- OAuth（Linux.do）：[oauth_handlers.go](../../backend/internal/api/oauth_handlers.go) + 前端 [OAuthCallback.vue](../../frontend/src/views/OAuthCallback.vue) + [AuthModal.vue](../../frontend/src/components/AuthModal.vue)

### H. 改存储
- OSS：[storage/oss.go](../../backend/internal/storage/oss.go)（换 bucket/region/域名）+ env `OSS_*`
- objectKey 规则和 URL 拼接在 `buildPublicURL` / `UploadImageData`

## 扩展点（项目已留好接口，优先用这些）

| 想加的功能 | 扩展点 |
|-----------|--------|
| 新 AI 模型 | `ImageGenerator`/`VideoProvider` 接口 + 注册表 |
| 新支付渠道 | [payment/provider.go](../../backend/internal/payment/provider.go) `PaymentProvider` 接口 + 在 payment_handlers 切换 |
| 新生成类型 | `UnifiedGenerateRequest.Type` 加分支（[generate_handlers.go](../../backend/internal/api/generate_handlers.go) switch） |
| 新积分流水类型 | `CreditTxType` 常量 + `recordCreditTransaction` |
| 管理后台新模块 | [frontend-admin](../../frontend-admin/src/components/AdminReviewPage.vue) 菜单已预留 `user_list`/`generation_list` + 后端 `/api/admin` 加路由 |
| 工具页 | [Tools.vue](../../frontend/src/views/Tools.vue) + 路由（参考 image-to-svg/reverse-prompt/image-convert） |

## 改造前的环境准备

```bash
# 后端
cd backend
cp .env.example .env   # 必填: DB_USER/DB_PASSWORD/DB_NAME/JWT_SECRET/GOOGLE_API_KEY；其余按需
# 至少配一个 AI 供应商 key（GOOGLE_API_KEY 或 ARK_API_KEY），否则对应模型 IsAvailable()=false

# 前端
cd frontend && npm install && npm run dev      # :5173，/api 代理到 :8092
cd frontend-admin && npm install && npm run dev # :5174
```

> JWT_SECRET 为空服务**直接 log.Fatal 起不来**；DB 三件套缺一同样 fatal。OSS 未配置只 warning（上传相关功能会失败但不阻断启动）。

---

文档到此结束。后续提需求时，直接说"照 [07-customization.md](07-customization.md) 场景 X 改 ..."，我能快速定位。如发现文档与代码不符，**以代码为准**并请顺手更新本目录。
