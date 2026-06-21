# 03 · 前端详解

两套独立 Vue 3 SPA：用户端（[frontend/](../../frontend/)，Naive UI）和管理后台（[frontend-admin/](../../frontend-admin/)，Ant Design Vue）。共享同一个后端 `:8092`，但**鉴权体系完全独立**（用户端 JWT / 管理后台 X-Admin-Token）。

## 1. 用户端启动 [main.js](../../frontend/src/main.js)

插件顺序：`createPinia()` → `router` → `i18n` → `naive`（全量 Naive UI）。

**关键**：挂载前先 `usePricingStore().fetchPricing()`，数据回来后才 `app.mount('#app')`——确保定价表先加载，组件不会读到空价格。`App.vue` 的 `<script setup>` 同步阶段调 `userStore.init()`（早于子组件 onMounted）。

**axios 约定**：项目**无全局 axios 实例/拦截器**。各处直接 `import axios` 手动拼 `Authorization: Bearer ${token}`（从 `localStorage.token` 读）。仅 `generation.js` 和 `notifications.js` 各自 `axios.create({ baseURL: '/api' })` + 私有请求拦截器。401 时 `user.fetchUserInfo()` 调 `logout()`，其他请求通常只 `console.error`。

`index.html` 含完整 SEO meta（标题/OpenGraph/Twitter/JSON-LD WebApplication+FAQPage）、Google Fonts、canonical。

## 2. 路由与布局

### 路由 [router/index.js](../../frontend/src/router/index.js)
全部懒加载。分两组：
- **独立顶层**（`meta.guest:true`）：`/`(Landing)、`/terms-of-service`、`/privacy-policy`、`/oauth/callback`
- **AppLayout 子路由**：`inspiration`、`inspiration/search`、`inspiration/:shareId`、`generate`、`assets`、`account`(requiresAuth)、`tools`、`tools/image-to-svg`、`tools/reverse-prompt`、`tools/image-convert`
- 兜底 `/:pathMatch(.*)*` → 重定向 `/`

**守卫逻辑**（`beforeEach`）：
- `applyRouteMeta(to)` 每次导航按 `meta.titleKey/descriptionKey/keywordKey` 走 i18n 写 `document.title` / meta 标签（SEO）
- `meta.requiresAuth && !isLoggedIn` → 跳 landing 并带 `redirect`
- 已登录访问 landing（无 invite 参数）→ 跳 inspiration
- 监听 `window` 的 `locale-changed` 事件重渲 meta

### 布局
- **AppLayout.vue** — 外壳：`<AppSidebar />` + `<router-view />`。桌面 flex 两栏；移动端（≤768px）改底部 56px 导航条。
- **AppSidebar.vue** — 导航项：inspiration / generate / assets / tools。底部：每日签到入口、钻石余额（点开定价）、社区二维码、通知（NDrawer）、用户头像（主题/语言/登出）。

## 3. Pinia Stores（8 个）

| Store | 关键 State | 关键 Actions/Computed |
|-------|-----------|----------------------|
| **user** | `isLoggedIn`、`currentUser`、`token`、各弹窗显隐 ref | computed: `userCredits`/`inviteCode`/`dailyCheckinAvailable`/`isLinuxDoUser`；actions: `init`/`fetchUserInfo`(401→logout)/`loginSuccess`/`logout`/`requireAuth`(未登录开 AuthModal)/`dailyCheckin`/`createPaymentOrder`/`getPaymentStatus`/`bindEmail` |
| **generation** | `generations`、`pendingResult`、`filters` | computed `groupedGenerations`(按 today/yesterday/week/older 分组)；actions: `load`/`loadMore`/`updateGeneration`(PUT)/`toggleFavorite`/`deleteGeneration`/`prependGeneration` |
| **models** | `models`、`defaultModelId` | computed `imageModels`/`availableModels`/`ensureModelId`；`loadModels`(按优先级 `['gemini-3.1-flash-image-preview','gemini-3-pro-image-preview','doubao-seedream-4-5']` 选默认) |
| **pricing** | `imagePricingRaw`、`videoPricingRaw`、`ecommerceRaw` | methods: `getImageCredits`/`getVideoCredits`(×1.2 if audio)/`getVeoCredits`/`getEcommerceCredits`/`fetchPricing` |
| **theme** | `themeMode`(默认 dark)、`systemDark`、`themeOverride` | `setThemeMode`/`forceTheme`(Landing 强制 dark)/`clearForceTheme`；watch 写 `<html data-theme>` + localStorage |
| **locale** | `locale` | `setLocale` → 同步 i18n + localStorage + `dispatchEvent('locale-changed')` |
| **composerDraft** | `draft` | `setRemixDraft`/`setReferenceDraft`/`setPromptDraft`/`consumeDraft`(取走清空) — 灵感页→生成页传参 |
| **notifications** | `notifications`、`unreadCount` | 私有 axios 实例 baseURL `/api/user`；`loadNotifications`/`markAsRead`/`markAllAsRead` |

> theme/locale 自行 watch 写 localStorage；user token 存 `localStorage.token`。

## 4. Composables

### useGenerate.js — 生成流程 + 轮询
- `generate(type, payload)` → POST `/api/generate`（30s timeout），返回 `{task_id}`
- `generateImage`/`generateEcommerce`/`generateVideo` 都是包装
- `pollTask(taskId, onUpdate)` / 别名 `pollVideoTask`：每 **3s** GET `/api/generations/{id}`，success→完成、failed→失败、generating/queued/running→进行中。`activePolls` ref 防重复
- `uploadImageToOSS(base64)` → POST `/api/user/upload/image`
- `optimizePrompt`（45s）/ `reversePrompt`（60s）

### useInspiration.js — 灵感社区 API
`authHeaders()` 已登录返回 Bearer 否则 `{}`（多数接口允许游客）。方法：`listInspirations`/`listLiked`/`listMine`/`getInspiration`/`getLikeStatus`/`like`/`unlike`/`markRemix`/`shareGeneration`/`publishInspiration`/`listInspirationTags`/`uploadVideo`/`unshareInspiration`。

## 5. 核心页面

| 页面 | 角色 |
|------|------|
| **Landing.vue** | 营销落地页，`forceTheme('dark')`，所有 CTA 跳 `/inspiration` |
| **Inspiration.vue** | 主探索 feed：瀑布流 + ComposerBar + 标签 tab + 发布对话框 + 无限滚动 + hover 自动播视频。维护 `inspirationPageCache` 恢复滚动 |
| **InspirationDetail.vue** | 单帖详情（`:shareId`）：媒体预览 + 作者 + 提示词复制 + 点赞/remix |
| **InspirationSearch.vue** | 搜索结果（读 `route.query.q`），watch 重新拉取 |
| **Generate.vue** ★ | **核心创作页**，见下方详解 |
| **Assets.vue** | 个人作品库（type/sub 过滤，含 liked/shared 子视图，收藏/分享/删除） |
| **Account.vue** | 签到/兑换/邀请记录/积分流水；监听 `visibilitychange` 刷新 |
| **Tools.vue** | 工具导航卡（reverse-prompt / image-to-svg / image-convert + coming soon） |
| **ReversePrompt.vue** | 上传图片反推提示词，402 时 `openPricing()` |
| **ImageToSvg.vue** | 位图转 SVG，**纯客户端**（imagetracerjs），无网络 |
| **ImageConvert.vue** | 格式/质量转换，**纯客户端**（canvas），无网络 |
| **OAuthCallback.vue** | LinuxDo OAuth 回调，popup 模式 postMessage 传 token |

### Generate.vue 详解（[views/Generate.vue](../../frontend/src/views/Generate.vue)）
布局：历史时间线（`genStore.groupedGenerations` 按日期分组）+ 顶部 `ComposerBar` + 当前会话临时结果 `currentResults`。

- **挂载**：`genStore.load(true)` → `resumePendingPolls()`（对 queued/running/generating 历史任务重启轮询）→ 消费 `genStore.pendingResult` → 消费 `composerDraftStore.consumeDraft()`（填回 ComposerBar）
- **三种 creativeMode**：`image` / `video` / `ecommerce`（与 ComposerBar 联动）
- **提交** `handleSubmit(payload)` → `executeGeneration()` 按 mode 构造 apiPayload → `generate()` → `startPoll(resultItem, task_id)`
- **轮询** success 时写回 `resultItem` + `userStore.fetchUserInfo()` + `genStore.prependGeneration`；failed 写 error_msg；`onUnmounted` → `stopAllPolls()`
- **Inpainting**：`editImage` 打开 ImageEditor → `handleInpaintSubmit` 先上传 mask 再 `type:'image', images:[原图], mask:maskUrl, model:'gemini-3-pro-image-preview'` 提交

## 6. 关键组件

| 组件 | 作用 |
|------|------|
| **ComposerBar.vue** | 统一输入条，覆盖 image/video/ecommerce 三模式。`defineExpose({fillPrompt, fillFromGeneration, fillEditImage})`。selectedModel 持久化 localStorage。视频参数（模型/分辨率/比例/时长，Veo 1080p/4k 强制 8s/generateAudio）、电商参数（平台/类型/张数 5-15）、提示词 AI 优化 |
| **ImageEditor.vue** | canvas 局部重绘编辑器。`lockedModel='gemini-3-pro-image-preview'`。笔刷/撤销栈(≤30 步)/比例分辨率。`exportMask()` 导出黑白 mask base64 |
| **AuthModal.vue** | 登录/注册/重置/兑换四模式。验证码 60s 倒计时，dev 显示 dev_code。LinuxDo OAuth 走 popup + postMessage。注册默认邀请码 `'VIP666DB'` |
| **PricingModal.vue** | 定价三档（见下）。LinuxDo 用户显示积分购买（预开 about:blank 防拦截 → createPaymentOrder → 轮询 getPaymentStatus 每 3s，超时 5min）。其他用户弹闲鱼卡 |
| **ShareGenerationDialog.vue** | 发布到灵感社区。图片多选 / 视频单传（自动抽帧封面），标题/描述/提示词(必填)/标签(≤5) |

**前端定价展示**（PricingModal，注意与后端套餐可能脱节，以 [pricing.go](../../backend/internal/api/pricing.go) 为准）：starter / popular（推荐）/ pro 三档。后端实际套餐金额见 [02-backend.md](02-backend.md)。

## 7. 管理后台 [frontend-admin/](../../frontend-admin/)

- **main.js** — 仅 `app.use(Antd)` + reset.css，**无 Pinia、无 vue-router**（单页）
- **App.vue** — `a-config-provider`（zhCN，主题 token）直接挂 `<AdminReviewPage />`
- **认证** — [useAdminInspiration.js](../../frontend-admin/src/composables/useAdminInspiration.js)：`ADMIN_TOKEN_KEY='admin_token'`，`adminHeaders()` 返回 `{ 'X-Admin-Token': token }`。登录即 `saveAdminToken` + `verifyToken()`（用 `listAdminInspirations({limit:1, review_status:'all'})` 探活）
- **AdminReviewPage.vue** — Ant 侧边栏，三模块菜单（`inspiration_review` 已实现 / `user_list`、`generation_list` 预留）。灵感审核台：筛选（状态/用户/关键词/日期）+ 分页表格 + 统计卡 + 行操作 approve/reject（reject 弹 prompt 填原因）。401/403/503 自动 logout

## 8. 构建与代理配置

### 用户端 vite.config.js
```js
server.proxy['/api'] = { target: 'http://localhost:8092', changeOrigin: true }  // 保留 /api 前缀
build.rollupOptions.output.manualChunks = { 'naive-ui': ['naive-ui'], 'vue-vendor': ['vue','vue-router','pinia'] }
```
- `/api` 代理到 `:8092`，无 `@` alias（全相对路径），未使用 `import.meta.env`

### 管理后台 vite.config.js
```js
server: { port: 5174, proxy: { '/api': { target: 'http://localhost:8092', changeOrigin: true } } }
```

### 依赖
- 用户端：vue 3 / vue-router 4 / pinia / naive-ui / vue-i18n / axios / @vicons/ionicons5 / imagetracerjs
- 管理后台：vue 3 / ant-design-vue / @ant-design/icons-vue / axios
