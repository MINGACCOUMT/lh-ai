# 02 · 后端详解

> 入口：[backend/main.go](../../backend/main.go)。本篇按"路由 → 中间件 → worker → 各 handler 文件 → 共享工具 → infra"组织。

## 1. 路由表（全部 `/api` 前缀）

### 公开（无鉴权）
| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/pricing` | `GetPricing` |
| GET | `/models` | `GetModels` |
| POST | `/auth/send-code` | `SendVerificationCode` |
| POST | `/auth/register` | `Register` |
| POST | `/auth/login` | `LoginWithEmail` |
| POST | `/auth/reset-password` | `ResetPassword` |
| GET | `/auth/oauth/linuxdo` | `LinuxDoOAuthURL` |
| POST | `/auth/oauth/linuxdo/callback` | `LinuxDoOAuthCallback` |
| GET | `/payment/notify/linuxdo` | `LinuxDoCreditNotify`（支付回调，验签） |
| GET | `/inspirations` | `ListPublicInspirations` |
| GET | `/inspiration-tags` | `ListInspirationTags` |

### 用户鉴权（`UserAuthMiddleware`）
| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/user/me` | `GetUserMe` |
| POST | `/user/redeem` | `RedeemKey` |
| POST | `/user/daily-checkin` | `DailyCheckin` |
| GET | `/user/invitations` | `GetInvitationRecords` |
| GET | `/user/credits/transactions` | `GetCreditTransactions` |
| GET | `/user/notifications` | `ListUserNotifications` |
| POST | `/user/notifications/read-all` | `MarkAllNotificationsRead` |
| POST | `/user/notifications/:id/read` | `MarkNotificationRead` |
| POST | `/user/bind-email` | `BindEmail` |
| POST | `/user/upload/image` | `UploadImage` |
| POST | `/user/upload/video` | `UploadVideo` |
| POST | `/user/payment/create` | `CreatePaymentOrder` |
| GET | `/user/payment/status/:orderNo` | `GetPaymentStatus` |
| GET | `/user/payment/orders` | `GetPaymentOrders` |
| POST | `/generate` | `UnifiedGenerate` ★ |
| POST | `/prompt/optimize` | `OptimizePrompt` |
| POST | `/tools/reverse-prompt` | `ReversePrompt` |
| GET | `/inspirations/liked` | `ListLikedInspirations` |
| GET | `/inspirations/mine` | `ListMyInspirations` |
| GET | `/inspirations/:shareID` | `GetPublicInspiration`（可选鉴权） |
| GET | `/inspirations/:shareID/liked` | `GetInspirationLikeStatus` |
| POST | `/inspirations/:shareID/like` | `LikeInspiration` |
| DELETE | `/inspirations/:shareID/like` | `UnlikeInspiration` |
| POST | `/inspirations/:shareID/remix` | `MarkInspirationRemix` |
| DELETE | `/inspirations/:shareID` | `UnshareInspirationByShareID` |
| POST | `/inspirations/publish` | `PublishInspiration` |

### 生成历史（`/generations`，用户鉴权）
| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/generations` | `ListGenerations` |
| GET | `/generations/:id` | `GetGeneration` |
| PUT | `/generations/:id` | `UpdateGeneration` |
| POST | `/generations/:id/share` | `ShareGeneration` |
| DELETE | `/generations/:id/share` | `UnshareGeneration` |
| DELETE | `/generations/:id` | `DeleteGeneration` |

### 管理员鉴权（`admin.AuthMiddleware`，`X-Admin-Token`）
| 方法 | 路径 | Handler |
|------|------|---------|
| GET | `/admin/inspirations` | `ListInspirations` |
| POST | `/admin/inspirations/:id/review` | `ReviewInspiration` |

> 完整入参/出参见 [05-api-reference.md](05-api-reference.md)。

## 2. 中间件

### `UserAuthMiddleware` — [middleware.go](../../backend/internal/api/middleware.go)
解析 `Authorization: Bearer <token>` → `auth.ValidateUserToken` → 查 User 存在且 `status=active` → 注入 `c.Set("userID", ...)` 和 `c.Set("email", ...)`。失败返回 401/403。

### `admin.AuthMiddleware` — [admin/middleware.go](../../backend/internal/api/admin/middleware.go)
读 `ADMIN_TOKEN` 环境变量（未配置返回 503）。token 取自 `X-Admin-Token` 头，否则 `Authorization: Bearer`。用 `subtle.ConstantTimeCompare` 防时序攻击。通过后注入 `adminOperatorSource="admin_console"`、`adminOperatorID=sha256(token)[:8]`。

## 3. 后台 Worker / 定时任务

| 函数 | 定义位置 | 间隔 | 作用 |
|------|----------|------|------|
| `StartVideoTaskPoller` | video_handlers.go | **10s** | 扫 `generations WHERE type='video' AND status IN ('queued','running','pending')`，每条 goroutine 调服务商查状态、转存视频到 OSS、**超 30 分钟自动 failed+退款**。`processingTasks sync.Map` 防同 task 并发 |
| `StartVerificationCleanup` | auth_handlers.go | **1h** | 删 `email_verifications` 中 `used=true OR expires_at<now` |
| `StartGenerationCleanup` | generation_handlers.go | **1h** | 删 30 天前 failed 的 generations + 3 天前 api_logs（**保留成功记录**） |
| OAuth state 清理 | oauth_handlers.go `init()` | **5m** | 清过期 OAuth state（CSRF） |

## 4. 各 Handler 文件职责

### auth_handlers.go — 邮箱账号体系
- `SendVerificationCode` — 发验证码（type ∈ register/login/reset/bind，1 分钟频率限制，10 分钟过期）。**注意：未配 SMTP 时 `dev_code` 实际不会返回**（[email.go](../../backend/internal/email/email.go) 未配置时返回 nil 不报错，而 handler 里 dev_code 只在 `err != nil` 分支返回 → 死代码路径），验证码只会打到后端日志 `邮件服务未配置，验证码 [xxxxxx]`
- `Register` — 注册：校验码 → 建用户(Credits=10) → 流水(register_gift) → **邀请奖励**（邀请人 +10 钻，累计上限 500，事务）→ 欢迎通知 → 发 JWT
- `LoginWithEmail` — 密码或验证码登录 → 更新 last_login → 发 JWT
- `ResetPassword` — 验证码重置密码
- `RedeemKey` — 兑换 License Key（JWT 解析 credits）→ 事务加余额 + 流水(redeem)
- `GetUserMe` — 返回用户全字段 + 签到状态计算
- `DailyCheckin` — **行锁**(clause.Locking UPDATE)防并发签到，连续天数判定，7 天循环奖励（day1-6=4+day 即 5~10，day7=15）
- `BindEmail` / `GetInvitationRecords` / `GetCreditTransactions`

### oauth_handlers.go — Linux.do OAuth
- `LinuxDoOAuthURL` — 生成 state（16 字节 hex，10 分钟有效，内存存储）→ 返回授权 URL
- `LinuxDoOAuthCallback` — 校验 state（一次性）→ 换 token → 拉 LinuxDo 用户信息 → 按 `linuxdo_id` 查/建用户（新建 Credits=10）→ 发 JWT

### generate_handlers.go — **统一生成入口** ★
`UnifiedGenerate` 按 `req.Type` 分发：
- `handleUnifiedImageGenerate` — 图片：扣费 → 建记录(generating) → **goroutine 异步**：下载输入图/mask → `provider.Get(model).GenerateImage` → 上传 OSS → 更新成功；失败 `updateGenerationFailed`（标 failed + 退款）
- `handleUnifiedVideoGenerate` — 视频：扣费 → 下载首帧/尾帧/参考图(≤3) → `videoProvider.CreateVideoTask` → 建记录(queued, TaskID)；状态由 poller 异步推进
- `handleUnifiedEcommerceGenerate` — 电商组图：扣费 → 上传输入图 → goroutine：`MultiImageGenerator.GenerateMultiImage` → 逐张上传 → **按实际产出数部分退款** → 更新成功

> 三条链路的扣费闭环一致，见下方"积分经济"。

### video_handlers.go — 视频任务轮询
`StartVideoTaskPoller` → `pollPendingVideoTasks` → `processVideoGeneration`：超时 30 分钟 failed+退款；成功时 `storage.DownloadAndUploadVideo` 转存（Veo 带 `x-goog-api-key` 头）；失败先更新 DB 再退款防重复。

### generation_handlers.go — 生成历史 CRUD
`ListGenerations`（分页 + type/favorite/shared 过滤，shared 通过 JOIN inspiration_posts 判定）、`GetGeneration`、`UpdateGeneration`（字段白名单）、`DeleteGeneration`（硬删）、`CreateGeneration`（**共享辅助函数，非 handler**：把 ReferenceImages/Params/Images 序列化 JSON 入库）。

### inspiration_handlers.go — 灵感广场
发布（`ShareGeneration`/`PublishInspiration`，支持 generation/upload 两种来源）、列表（公开/点赞/我的）、单条（view_count+1）、点赞（事务 + OnConflict DoNothing，防 like_count 负数）、取消分享（改 status=hidden）、remix 计数、标签列表。审核快照：`isInspirationAutoApprove()` 决定新帖 `approved`/`pending`（**注意默认值，见改造指引**）。

### prompt_optimize_handlers.go — 提示词优化（DeepSeek）
`OptimizePrompt` — 扣 `PROMPT_OPTIMIZE_CREDITS`（默认 1）→ 调 DeepSeek（temperature 0.35, response_format=json_object）→ 返回 candidates。失败退款。

### reverse_prompt_handlers.go — 反推提示词（豆包多模态）
`ReversePrompt` — 扣 2 钻（`reversePromptCredits=2`）→ 调 `doubao-seed-2-0-pro-260215` 多模态（image_url+text）→ 返回提示词。失败退款。

### resource_handlers.go — 上传
`UploadImage`（base64 → OSS）、`UploadVideo`（multipart，限 100MB，mp4/mov/webm/m4v）。

### notification_handlers.go — 站内通知
列表（带 unread_count）、标记已读（单条/全部）。

### model_handlers.go — 模型列表
`GetModels` — `provider.ListAvailable()` 套友好名返回。

## 5. 共享工具 [utils.go](../../backend/internal/api/utils.go)

| 函数 | 作用 |
|------|------|
| `getActiveUser(c, userID)` | 查用户 + 校验 active，失败自动写 HTTP 响应 |
| `recordCreditTransaction(tx, userID, delta, type, source, sourceID, note)` | 写积分流水。**前提：余额已先更新**（重新 Select 读最新余额写 BalanceAfter） |
| `refundCredits(userID, credits, reason)` | 统一退款（事务：加余额 + 写 refund 流水；`parseRefundSource` 解析 reason 前缀） |
| `logAPICall(...)` | 异步 goroutine（defer recover）写 APILog，脱敏大字段 |
| `sanitizeRequestBody` | 递归移除日志中的 image/images 等大字段 |

**积分流水类型常量**（CreditTxType）：`register_gift` / `invite_reward` / `redeem` / `generate_cost` / `prompt_optimize_cost` / `reverse_prompt_cost` / `refund` / `daily_checkin` / `online_payment` / `inspiration_review_reward`。

## 6. 积分经济（核心闭环）

所有扣费遵循统一范式（以图片为例）：

```
1. getActiveUser → user.Credits 校验余额
2. 原子扣费: UPDATE users SET credits=credits-?, usage_count+1 WHERE id=? AND credits>=?
   （RowsAffected==0 视为余额不足/扣费失败，返回 402）
3. recordCreditTransaction(负 delta, type=generate_cost)  ← 写流水
   （流水写入失败 → refundCredits 回滚，返回 500）
4. CreateGeneration(status=generating)
5. 异步生成；任一步失败 → updateGenerationFailed(标 failed + refundCredits)
6. 电商特殊：按实际产出张数重算 actualCreditsSpent，差额 refundCredits(ecommerce-partial-refund)
```

**退款幂等**：视频任务失败退款**先更新 DB 状态再退款**，避免 poller 重复触发；支付履约 `fulfillPaymentOrder` 用行锁 + 非 pending 短路；审核发奖按 credit_transaction 查重 + 通知表 OnConflict 双重防重。

## 7. Infra

### storage/oss.go — 阿里云 OSS
`InitOSS` / `UploadImageData` / `UploadBase64Image` / `UploadVideoData` / `DownloadAndUploadVideo`（300s 超时，支持 HTTP_PROXY，可选请求头供 Veo 用）。objectKey 规则：图 `<dir>/<ms>_<hex>.png`，视频 `videos/<uid>/<ms>_<hex>.mp4`。公开 URL 优先 `OSS_PUBLIC_DOMAIN`。

### email/email.go — SMTP
`InitEmail` / `IsConfigured` / `SendVerificationCode`（按 purpose 不同 HTML 模板）。**QQ 邮箱 `"short response"` 错误视为成功**（Go 已知 issue #24845）。未配置时仅日志返回 nil。

### payment/ — 支付
- `provider.go` — `PaymentProvider` 接口（CreatePayment/VerifyNotify/QueryOrder）
- `linuxdo_credit.go` — Linux.do Credit（EPay 协议）。端点 `credit.linux.do/epay/pay/submit.php`（下单）、`/epay/api.php`（查询）。签名：过滤空值与 sign/sign_type → key ASCII 升序拼 `k=v&` → 末尾加 secret → MD5 小写。**VerifyNotify 非恒定时间比较**（已知小瑕疵）。

### cmd/ — CLI 工具
- `keygen/main.go` — `go run ./cmd/keygen -credits 200` 生成 License 兑换密钥（JWT 形式，claims 含 ID+Credits）
- `import_inspiration_full_json/main.go` — 从 JSON 批量导入灵感帖（share_id=`fj`+sha1(mediaURL)[:14]，upsert + 标签）
