# 05 · API 接口参考

> 所有接口前缀 `/api`。响应：成功直接返回业务 JSON，失败 `{"error":"<中文信息>"}`。完整路由见 [02-backend.md](02-backend.md)，DTO 定义见 [types.go](../../backend/internal/api/types.go)。

## 鉴权
- 用户端：`Authorization: Bearer <jwt>`（7 天有效期，见 [auth.go](../../backend/internal/auth/auth.go)）
- 管理后台：`X-Admin-Token: <token>`（静态，`subtle.ConstantTimeCompare` 校验）

---

## 公开接口

### GET `/api/pricing`
完整定价表。响应：
```json
{
  "image": [{ "model": "gemini-3-pro-image-preview", "model_name": "🍌 Nanobanana Pro", "description": "基于 Gemini 3 Pro",
              "prices": [{"size":"1K","credits":10,"description":"1024x1024"}, ...] }, ...],
  "video": { "base_per_second": {"480p":6,"720p":10,"1080p":16}, "audio_multiplier": 1.2, "veo_base_per_second": {"720p":45,"1080p":65,"4k":90} },
  "ecommerce": { "model": "doubao-seedream-4-5", "model_name": "Seedream-4.5", "prices": {"2K":6,"4K":10} },
  "exchange_rate": "1=10"
}
```

### GET `/api/models`
`{ "models": [{ "id", "name", "provider", "description", "available" }] }`

### POST `/api/auth/send-code`
入参：`{ "email": "...", "type": "register|login|reset|bind" }`。邮件未配置时返回 `{ "dev_code": "123456" }`（开发模式）。

### POST `/api/auth/register`
入参：`{ "email", "code"(6位), "password"(≥6位), "nickname?", "invite_code?" }`
响应：`{ "message", "token", "user": { id, email, nickname, credits, invite_code } }`（新建用户初始 10 钻）

### POST `/api/auth/login`
入参：`{ "email", "password"?, "code"? }`（密码或验证码二选一）
响应：同 register 的 token+user 结构

### POST `/api/auth/reset-password`
入参：`{ "email", "code", "password" }`

### OAuth
- GET `/api/auth/oauth/linuxdo` → `{ "url": "..." }`
- POST `/api/auth/oauth/linuxdo/callback` → 入参 `{ "code", "state" }`，响应 token+user

### GET `/api/payment/notify/linuxdo`
支付回调（query 参数）。验签 → 履约（幂等）→ 返回 success 字符串。

### GET `/api/inspirations`
query：`limit`, `offset`, `type?`, `tag?`, `q?`。响应：`{ "items": [InspirationPostResponse], "total", "limit", "offset" }`

### GET `/api/inspiration-tags`
`[{ "name", "slug" }]`（按 usage_count DESC）

---

## 用户接口（JWT）

### GET `/api/user/me`
返回用户全字段 + 签到状态：`{ ..., "daily_checkin_available": bool, "checkin_streak": int, "next_checkin_reward": int, "is_linuxdo": bool, "email_verified": bool }`

### POST `/api/user/redeem`
入参 `{ "key": "<LicenseKey>" }`。兑换钻石到余额。

### POST `/api/user/daily-checkin`
事务行锁防并发。返回签到奖励。已签到当日返回 409。

### GET `/api/user/invitations` · GET `/api/user/credits/transactions`
query：`limit`, `offset`, `type?`。流水返回 `{ delta, balance_after, type, source, source_id, note, created_at }`。

### 通知
- GET `/api/user/notifications` → `{ items, unread_count, total }`
- POST `/api/user/notifications/read-all` → `{ updated }`
- POST `/api/user/notifications/:id/read`

### POST `/api/user/bind-email`
入参 `{ "email", "code" }`（type=bind 验证码）。

### 上传
- POST `/api/user/upload/image` → 入参 `{ "image": "<base64>" }`，响应 `{ "url" }`
- POST `/api/user/upload/video` → multipart `file`（≤100MB，mp4/mov/webm/m4v），响应 `{ "url" }`

### 支付
- POST `/api/user/payment/create` → 入参 `{ "plan": "starter|popular|pro" }`（**仅 Linux.do 用户**），响应 `{ "order_no", "payment_url" }`
  - 套餐（[payment_handlers.go](../../backend/internal/api/payment_handlers.go)）：`starter ¥129/100钻`、`popular ¥559/500钻`、`pro ¥999/1000钻`
- GET `/api/user/payment/status/:orderNo` → `{ order_no, status, diamonds, plan_name, amount, paid_at }`（pending 时主动查服务商）
- GET `/api/user/payment/orders` → 分页订单列表

---

## 生成（JWT）

### POST `/api/generate` ★
入参（`UnifiedGenerateRequest`）：
```json
{
  "type": "image|video|ecommerce",   // required
  "prompt": "...",
  "images": ["url|base64", ...],      // 参考图（image≤3 / video≤3 / ecommerce 1~3）
  "mask": "url|base64",               // 仅 image，局部重绘
  "model": "模型ID?",
  "params": { ... }                   // 见下，按 type 不同
}
```
**params 按 type**：
- image：`aspectRatio`("1:1"等，默认 1:1)、`imageSize`("1K/2K/4K"，默认 2K)
- video：`mode`(text-to-video/first-frame/first-last-frame)、`resolution`(480p/720p/1080p)、`ratio`、`duration`、`generate_audio`、`first_frame`、`last_frame`
- ecommerce：`aspectRatio`、`imageSize`、`outputCount`(5~15 默认 7)、`imageType`、`ecommerceType`

响应（图片/电商 202，视频 200）：
```json
{ "task_id": <genID>, "status": "generating|queued", "credits_spent": N, "credits_remaining": N, "provider_task_id": "..." }
```
余额不足返回 402：`{ "error":"钻石不足", "required_credits", "current_balance" }`。

### POST `/api/prompt/optimize`
入参（`PromptOptimizeRequest`）：`{ "prompt"(≤4000), "creative_mode"?(image/video/ecommerce), "style"?(balanced/creative/detail/commercial), "target_model"?, "current_params"? }`
响应：`{ "raw_prompt", "candidates": [{ id, title, prompt, reason }], "meta": {...} }`。扣 `PROMPT_OPTIMIZE_CREDITS`（默认 1）。

### POST `/api/tools/reverse-prompt`
入参 `{ "image"(base64), "language"?(默认 zh), "target_model"?(默认 Nanobanana Pro) }`。响应 `{ "prompt", "meta": {...} }`。扣 2 钻。

---

## 生成历史（JWT，`/api/generations`）

- GET `` → query `limit`, `offset`, `type?`, `favorite?`, `shared?` → `{ items: [GenerationResponse], total, ... }`
- GET `/:id` → `GenerationResponse`
- PUT `/:id` → `UpdateGenerationRequest`（images/video_url/status/credits_cost/error_msg/task_id/is_favorite）
- POST `/:id/share` → `ShareGenerationRequest`：`{ title(≤200), description(≤1000), prompt?, tags?[]≤5, cover_url? }`
- DELETE `/:id/share` · DELETE `/:id`（硬删）

**GenerationResponse** 字段：`id, type, prompt, reference_images[], params{}, images[], video_url, status, credits_cost, error_msg, task_id, is_favorite, is_shared, share_id, created_at(UnixMilli), updated_at`

---

## 灵感广场

- GET `/api/inspirations`（公开，可选鉴权）/ `/liked`（JWT）/ `/mine`（JWT）
- GET `/api/inspirations/:shareID`（公开，view_count+1，可选鉴权返回 is_liked）
- GET `/api/inspirations/:shareID/liked`（JWT）→ `{ "liked": bool }`
- POST `/api/inspirations/:shareID/like`（JWT）/ DELETE `.../like`（JWT）
- POST `/api/inspirations/:shareID/remix`（公开匿名，remix_count+1）
- DELETE `/api/inspirations/:shareID`（JWT，作者取消分享 → status=hidden）
- POST `/api/inspirations/publish`（JWT）→ `PublishInspirationRequest`：`{ source_type?, generation_id?, type?, title(≤200), description(≤1000), prompt?, tags?, images?, video_url?, cover_url?, reference_images?, params? }`

**InspirationPostResponse** 字段：`id, share_id, type, source_type, title, description, prompt, tags[], params{}, reference_images[], images[], video_url, cover_url, source_generation_id, view_count, like_count, remix_count, review_status, reviewed_*, is_liked, published_at, author{user_id,nickname,avatar}`

---

## 管理后台（X-Admin-Token，`/api/admin`）

- GET `/api/admin/inspirations` → query `limit`, `offset`, `review_status`(pending/all/approved/rejected，默认 pending), `user_id?`, `start_date?`, `end_date?`, `q?` → `{ items, total, limit, offset, review_status }`
- POST `/api/admin/inspirations/:id/review` → `{ "action": "approve|reject", "note": "?" }`。approve 时发审核奖励（video+4 钻 / 其他+2 钻，幂等）+ 通知。
