# 04 · 数据模型

> 模型定义：[backend/internal/db/db.go](../../backend/internal/db/db.go)。迁移机制：**自定义 SQL 文件运行器**（`runMigrations`），**不使用 GORM AutoMigrate**。

## 迁移机制（重要）

`db.InitDB()` 连接 MySQL 后调用 `runMigrations()`：

1. 建 `schema_migrations(version, applied_at)` 跟踪表（如不存在）
2. 定位 `migrations/` 目录（候选 `./migrations` 或 `./backend/migrations`）
3. 收集 `.sql` 文件，**按文件名升序**排序
4. 读已应用版本（`SELECT version FROM schema_migrations`）
5. 对未应用的文件：`splitSQL` 按 `;` 分割（跳过纯注释行）→ 逐条 `DB.Exec` → 全部成功才记入 `schema_migrations`

**改表结构 = 新增 `migrations/NNN_描述.sql`**（编号递增，跟在 026 之后）。`splitSQL` 简单按 `;` 分割，**不支持存储过程/字符串内含分号**，写迁移时注意。

## 表清单（13 张 + 跟踪表）

### `users` — 用户
| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint64 PK | 用户 ID |
| email | varchar(255) unique | 邮箱（OAuth 用户可能为占位） |
| password_hash | varchar(255) | bcrypt 哈希（`json:"-"` 不外泄） |
| linuxdo_id | varchar(100) unique nullable | Linux.do OAuth ID |
| nickname / avatar | varchar | 昵称 / 头像 URL |
| credits | int default 0 | **可用钻石余额** |
| total_redeemed | int | 累计兑换钻石 |
| usage_count | int | 总使用次数 |
| status | varchar(20) default 'active' | 非 active 被禁用 |
| email_verified | bool | 邮箱是否验证 |
| invite_code | varchar(20) unique | 自己的邀请码 |
| invited_by | bigint nullable | 邀请人 ID |
| invite_count | int | 已邀请人数 |
| checkin_streak | int | 连续签到天数 |
| last_checkin_date | date nullable | 上次签到日期 |
| last_login_at | datetime nullable | |
| created_at / updated_at | datetime | |

### `licenses` — 兑换密钥（License Key）
| 字段 | 说明 |
|------|------|
| id | varchar(36) PK（UUID） |
| balance / status(active/disabled/redeemed) / expires_at / usage_count | |
| redeemed_by(uint64) / redeemed_at / original_key | |
> keygen 工具生成 JWT 形式密钥，claims 含 ID + Credits；兑换时解析 credits 加到用户余额。

### `email_verifications` — 验证码
id / email(index) / code(6位) / type(register/login/reset/bind) / expires_at(10分钟) / used / created_at

### `credit_transactions` — **积分流水账**（核心）
| 字段 | 说明 |
|------|------|
| id / user_id(index) | |
| delta | int（正=充入，负=扣除） |
| balance_after | int（当时余额快照） |
| type | varchar(40) index：`register_gift`/`invite_reward`/`redeem`/`generate_cost`/`prompt_optimize_cost`/`reverse_prompt_cost`/`refund`/`daily_checkin`/`online_payment`/`inspiration_review_reward` |
| source | varchar(40)：`license`/`generate`/`linuxdo_credit`/`inspiration_review` 等 |
| source_id | varchar(100)：关联业务 ID |
| note / created_at | |

### `user_notifications` — 站内通知
id / user_id / biz_key(index, 幂等) / title / summary / content / is_read(index) / created_at / updated_at

### `invitation_records` — 邀请记录
id / inviter_id / invitee_id / invitee_email / credits_rewarded(default 10) / created_at

### `generations` — **统一生成历史**（核心）
| 字段 | 说明 |
|------|------|
| id / user_id(index) | |
| type | varchar(20) default 'image'：image / video |
| prompt | longtext |
| reference_images | text（JSON：输入图 URL 数组） |
| params | text（JSON：模型/分辨率/比例/时长等） |
| images | text（JSON：输出图 URL 数组） |
| video_url | varchar(500)（视频输出） |
| status | varchar(20) default 'success'：generating/queued/running/success/failed |
| credits_cost | int |
| error_msg | text |
| task_id | varchar(100) nullable index（视频服务商任务 ID） |
| is_favorite | bool index |
| created_at / updated_at | |

### `image_records` / `image_templates` — 遗留兼容表
`image_records` 注释为 "legacy and kept for compatibility"。新功能用 `generations`。

### `payment_orders` — 在线支付订单
id / user_id / order_no(varchar(64) unique) / provider / provider_trade_no / amount(varchar) / diamonds / plan_name / status(default 'pending') / notify_data / paid_at / created_at / updated_at

### `inspiration_posts` — 灵感帖（社区广场）
| 字段 | 说明 |
|------|------|
| id / share_id(varchar(40) unique) | 公开分享 ID（12 位随机） |
| user_id(index) / source_generation_id(unique nullable) | 来源 generation（可空，上传发布时为空） |
| source_type | generation / upload |
| type | image / video |
| title / description / prompt / params / reference_images | |
| media_urls | text（JSON：媒体 URL 数组） |
| cover_url | varchar(1000) 卡片封面 |
| status | published / hidden（取消分享=hidden） |
| review_status | approved / pending / rejected |
| reviewed_by_source / reviewed_by_id / reviewed_at | 审核快照 |
| view_count / like_count / remix_count | |
| published_at(index) / created_at / updated_at | |

### `inspiration_likes` — 点赞关系
id / user_id / post_id / created_at（联合语义：一个用户对一个帖一条）

### `inspiration_tags` — 标签字典
id / name(unique) / slug(unique) / status(default 'active') / usage_count / created_at / updated_at

### `inspiration_post_tags` — 帖子-标签关联
id / post_id / tag_id / created_at

### `inspiration_review_logs` — 审核日志
id / post_id / action / from_status / to_status / note / operator_source / operator_id / created_at

### `api_logs` — API 调用日志
id / user_id / endpoint / request_body / response_body / response_code / duration_ms / created_at（3 天自动清理）

### `schema_migrations` — 迁移跟踪
version(PK) / applied_at

## 迁移脚本索引（26 个）

| 文件 | 用途 |
|------|------|
| 001_add_user_system | 初始化用户系统（users/licenses/api_logs/email_verifications） |
| 002_add_invitation_system | 邀请系统 |
| 003_rename_api_logs_license_id | api_logs.license_id → user_id |
| 004_add_video_tasks | 视频任务表（后被 010 删除） |
| 005_fix_provider_response_type | provider_response → LONGTEXT |
| 006_add_conversations | 会话+消息表（后被 009 移除） |
| 007_change_message_task_id_to_string | messages.task_id → VARCHAR(200) |
| 008_add_messages_task_id_index | 加索引 |
| 009_refactor_to_generations | 重构为单一 generations 表 |
| 010_drop_legacy_tables | 删 video_tasks |
| 011_add_credit_transactions | 积分流水表 |
| 012_drop_redeem_histories | 删旧 redeem_histories |
| 013_add_inspiration_posts | 灵感帖表 |
| 014_add_inspiration_likes | 点赞表 |
| 015_drop_inspiration_video_cover_url | 删废弃列 |
| 016_drop_generations_video_cover_url | 删废弃列 |
| 017_decouple_inspiration_posts_from_generations | 灵感帖独立快照（源 generation 删后保留） |
| 018_add_inspiration_posts_title_description | 加 title/description |
| 019_upgrade_inspiration_publish_schema | 用户上传发布 + 规范化标签 |
| 020_refactor_inspiration_media_urls | 简化为 media_urls(JSON)+type+cover_url |
| 021_finalize_inspiration_review_schema | 审核快照字段定型 |
| 022_add_user_notifications | 通知表 |
| 023_drop_generations_tags | 删兼容列 |
| 024_add_checkin_fields | users 加签到字段 |
| 025_add_oauth_fields | users 加 linuxdo_id |
| 026_add_payment_orders | 支付订单表 |

> 004/006 等已被后续迁移删除/重构，但 .sql 文件保留（迁移历史不可删）。
