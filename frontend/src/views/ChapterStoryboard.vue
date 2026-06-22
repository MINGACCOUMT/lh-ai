<template>
  <div class="chapter-storyboard-page">
    <!-- Topbar -->
    <div class="storyboard-topbar">
      <div class="topbar-left">
        <button class="back-btn" @click="router.back()">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6" />
          </svg>
          <span>{{ t('novel.title') }}</span>
        </button>
        <div class="topbar-title">
          <span class="topbar-name">{{ chapter?.title || '…' }}</span>
          <span
            v-if="analysisStatus && analysisStatus !== 'none'"
            class="status-badge"
            :class="`status-${analysisStatus}`"
          >
            {{ analysisStatusText }}
          </span>
        </div>
      </div>
    </div>

    <!-- Body -->
    <div class="storyboard-scroll">
      <n-spin :show="loading && !chapter">
        <!-- Analyze prompt -->
        <div v-if="chapter && analysisStatus !== 'ready'" class="analyze-block">
          <div class="analyze-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 3v4M3 5h4M6 17v4M4 19h4" />
              <path d="M13 3l2.5 6.5L22 12l-6.5 2.5L13 21l-2.5-6.5L4 12l6.5-2.5L13 3z" />
            </svg>
          </div>
          <div class="analyze-info">
            <p class="analyze-title">{{ analysisStatus === 'failed' ? '解析失败，请重试' : '解析本章' }}</p>
            <p class="analyze-desc">
              {{ analysisStatus === 'failed'
                ? '提取大纲、人物画像与关键场景失败，请重新尝试。'
                : '提取本章大纲、人物画像与关键场景，用于生成分镜。' }}
            </p>
          </div>
          <button
            class="generate-btn"
            :class="{ busy: analysisStatus === 'analyzing' }"
            :disabled="analysisStatus === 'analyzing'"
            @click="onAnalyze"
          >
            <svg v-if="analysisStatus === 'analyzing'" class="btn-icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 12a9 9 0 1 1-6.219-8.56" />
            </svg>
            <svg v-else class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M5 3v4M3 5h4M6 17v4M4 19h4" />
              <path d="M13 3l2.5 6.5L22 12l-6.5 2.5L13 21l-2.5-6.5L4 12l6.5-2.5L13 3z" />
            </svg>
            {{ analysisStatus === 'analyzing' ? '解析中…' : (analysisStatus === 'failed' ? '重新解析' : '解析本章') }}
          </button>
        </div>

        <!-- Analysis cards -->
        <div v-if="analysisStatus === 'ready'" class="analysis-grid">
          <section v-if="chapter?.outline" class="info-card">
            <header class="info-header">
              <span class="info-dot"></span>
              <span class="info-title">本章大纲</span>
            </header>
            <p class="info-text">{{ chapter.outline }}</p>
          </section>
          <section v-if="chapter?.characters" class="info-card">
            <header class="info-header">
              <span class="info-dot"></span>
              <span class="info-title">人物画像</span>
            </header>
            <p class="info-text">{{ chapter.characters }}</p>
          </section>
          <section v-if="chapter?.scenes" class="info-card">
            <header class="info-header">
              <span class="info-dot"></span>
              <span class="info-title">关键场景</span>
            </header>
            <p class="info-text">{{ chapter.scenes }}</p>
          </section>
        </div>

        <!-- Assets section -->
        <div v-if="analysisStatus === 'ready'" class="assets-section">
          <div class="assets-header">
            <div class="assets-title-wrap">
              <span class="assets-title-dot"></span>
              <span class="assets-title">人物 / 场景图库</span>
              <span
                v-if="assetsStatus !== 'none'"
                class="status-badge"
                :class="`status-${assetsStatus}`"
              >
                {{ assetsStatusText }}
              </span>
            </div>
            <button
              class="generate-btn small"
              :class="{ busy: assetsStatus === 'generating' }"
              :disabled="assetsStatus === 'generating'"
              @click="onGenerateAssets"
            >
              <svg v-if="assetsStatus === 'generating'" class="btn-icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 12a9 9 0 1 1-6.219-8.56" />
              </svg>
              <svg v-else class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
              {{ assetsStatus === 'generating' ? '生成中…' : (assetsStatus === 'failed' ? '重新生成' : '生成人物/场景图') }}
            </button>
          </div>

          <template v-if="assetsReady">
            <!-- Pool toggle: novel library vs public pool -->
            <div class="asset-pool-toggle">
              <button
                class="pool-tab"
                :class="{ active: activePool === 'novel' }"
                @click="activePool = 'novel'"
              >📖 小说资源库</button>
              <button
                class="pool-tab"
                :class="{ active: activePool === 'public' }"
                @click="onSelectPublic"
              >🌍 公共池</button>
            </div>

            <div v-if="assetCharacters.length" class="asset-group">
              <div class="asset-group-label">角色库</div>
              <div class="asset-grid">
                <div v-for="c in assetCharacters" :key="c.id" class="asset-card">
                  <img v-if="assetImage(c)" :src="assetImage(c)" class="asset-img" :alt="c.novel_asset_name" />
                  <div v-else class="asset-img-placeholder">无图</div>
                  <div class="asset-info">
                    <p class="asset-name">{{ c.novel_asset_name || '—' }}</p>
                    <p v-if="c.prompt" class="asset-desc">{{ c.prompt }}</p>
                    <button
                      class="asset-move-btn"
                      @click="onMoveAsset(c)"
                    >{{ activePool === 'novel' ? '移到公共池' : '移到本书' }}</button>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="assetScenes.length" class="asset-group">
              <div class="asset-group-label">场景库</div>
              <div class="asset-grid">
                <div v-for="s in assetScenes" :key="s.id" class="asset-card">
                  <img v-if="assetImage(s)" :src="assetImage(s)" class="asset-img" :alt="s.novel_asset_name" />
                  <div v-else class="asset-img-placeholder">无图</div>
                  <div class="asset-info">
                    <p class="asset-name">{{ s.novel_asset_name || '—' }}</p>
                    <p v-if="s.prompt" class="asset-desc">{{ s.prompt }}</p>
                    <button
                      class="asset-move-btn"
                      @click="onMoveAsset(s)"
                    >{{ activePool === 'novel' ? '移到公共池' : '移到本书' }}</button>
                  </div>
                </div>
              </div>
            </div>
          </template>
          <div v-else-if="assetsStatus === 'ready'" class="assets-empty">
            <NEmpty size="small" description="暂无人物/场景图" />
          </div>
        </div>

        <!-- Plots -->
        <div v-if="analysisStatus === 'ready' && plots.length" class="plot-list">
          <article v-for="plot in plots" :key="plot.id" class="plot-card">
            <header class="plot-header">
              <div class="plot-title-wrap">
                <span class="plot-index">{{ plot.plot_index }}</span>
                <span class="plot-title-text">{{ plot.title || '—' }}</span>
              </div>
              <div class="plot-actions">
                <span
                  v-if="plot.storyboard_status && plot.storyboard_status !== 'none'"
                  class="status-badge"
                  :class="`status-${plot.storyboard_status}`"
                >
                  {{ storyboardStatusText(plot.storyboard_status) }}
                </span>
                <button
                  v-if="plot.storyboard_status !== 'ready'"
                  class="generate-btn small"
                  :class="{ busy: plot.storyboard_status === 'extracting' }"
                  :disabled="plot.storyboard_status === 'extracting'"
                  @click="onStoryboard(plot)"
                >
                  <svg v-if="plot.storyboard_status === 'extracting'" class="btn-icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 12a9 9 0 1 1-6.219-8.56" />
                  </svg>
                  <svg v-else class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M5 3v4M3 5h4M6 17v4M4 19h4" />
                    <path d="M13 3l2.5 6.5L22 12l-6.5 2.5L13 21l-2.5-6.5L4 12l6.5-2.5L13 3z" />
                  </svg>
                  {{ plot.storyboard_status === 'extracting' ? '生成中…' : (plot.storyboard_status === 'failed' ? '重新生成' : '生成分镜') }}
                </button>
              </div>
            </header>
            <p v-if="plot.summary" class="plot-summary">{{ plot.summary }}</p>

            <!-- Shots sub-grid -->
            <div v-if="plot.storyboard_status === 'ready' && plot.shots?.length" class="shots-grid">
              <article v-for="s in plot.shots" :key="s.id" class="shot-card">
                <header class="shot-header">
                  <span class="shot-index">分镜 {{ s.shot_index }}</span>
                </header>
                <div class="shot-body">
                  <label class="shot-field">
                    <span class="shot-field-label">场景</span>
                    <n-input v-model:value="s.scene" type="textarea" :autosize="{ minRows: 1 }" placeholder="场景" @update:value="scheduleSave(s)" />
                  </label>
                  <label class="shot-field">
                    <span class="shot-field-label">人物</span>
                    <n-input v-model:value="s.characters" placeholder="人物" @update:value="scheduleSave(s)" />
                  </label>
                  <label class="shot-field">
                    <span class="shot-field-label">图片提示词</span>
                    <n-input v-model:value="s.prompt" type="textarea" :autosize="{ minRows: 2 }" placeholder="图片提示词" @update:value="scheduleSave(s)" />
                  </label>
                  <label class="shot-field">
                    <span class="shot-field-label">对白 / 旁白</span>
                    <n-input v-model:value="s.dialogue" type="textarea" :autosize="{ minRows: 1 }" placeholder="对白 / 旁白" @update:value="scheduleSave(s)" />
                  </label>
                  <label class="shot-field">
                    <span class="shot-field-label">镜头运动</span>
                    <n-input v-model:value="s.camera" placeholder="镜头运动" @update:value="scheduleSave(s)" />
                  </label>
                </div>
              </article>
            </div>
            <div v-else-if="plot.storyboard_status === 'ready'" class="plot-empty">
              <NEmpty size="small" description="该情节暂无分镜" />
            </div>
          </article>
        </div>

        <!-- Empty -->
        <div v-if="chapter && analysisStatus === 'ready' && !plots.length" class="storyboard-empty-wrap">
          <NEmpty description="暂无情节" />
        </div>
      </n-spin>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NEmpty, NSpin, NInput, useMessage } from 'naive-ui'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const message = useMessage()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

const loading = ref(false)
const activePool = ref('novel') // 'novel' | 'public'
let pollTimer = null
const saveTimers = new Map()

const chapter = computed(() => store.chapterDetail?.chapter || null)
const plots = computed(() => store.chapterDetail?.plots || [])
const analysisStatus = computed(() => chapter.value?.analysis_status || 'none')

const analysisStatusText = computed(() => statusText(analysisStatus.value))

const assetsStatus = computed(() => chapter.value?.assets_status || 'none')

// Active pool source: novel-scoped store.assets, or public store.publicAssets
const currentAssets = computed(() => activePool.value === 'public' ? store.publicAssets : store.assets)
const assetsReady = computed(() => {
  if (activePool.value === 'public') {
    return !!(store.publicAssets?.characters?.length || store.publicAssets?.scenes?.length)
  }
  return assetsStatus.value === 'ready' && !!(store.assets?.characters?.length || store.assets?.scenes?.length)
})
const assetCharacters = computed(() => currentAssets.value?.characters || [])
const assetScenes = computed(() => currentAssets.value?.scenes || [])
const assetsStatusText = computed(() => statusText(assetsStatus.value))

function assetImage(asset) {
  try { return JSON.parse(asset.images || '[]')[0] || '' } catch { return '' }
}

async function onSelectPublic() {
  activePool.value = 'public'
  if (!store.publicAssets) {
    try { await store.loadPublicAssets() } catch { message.error('加载公共池失败') }
  }
}

async function onMoveAsset(asset) {
  const targetId = activePool.value === 'novel' ? null : (chapter.value?.novel_id || null)
  try {
    await store.moveAssetScope(asset.id, targetId)
    message.success(activePool.value === 'novel' ? '已移到公共池' : '已移到本书')
  } catch (e) {
    message.error(e.response?.data?.error || '移动失败')
  }
}

onMounted(async () => {
  await loadChapterData(route.params.cid)
})

// Vue Router reuses this component across chapter-id changes, so onMounted
// won't re-fire on navigation. Reload (and reset local UI state) when the cid
// param changes. First load is handled by onMounted above.
watch(
  () => route.params.cid,
  (newCid, oldCid) => {
    if (newCid && newCid !== oldCid) loadChapterData(newCid)
  }
)

// Shared loader used by both onMounted and the route watcher. Resets all
// local UI state so nothing from the previous chapter leaks through.
async function loadChapterData(cid) {
  // Cancel any in-flight polling and pending shot saves for the old chapter.
  if (pollTimer) { clearTimeout(pollTimer); pollTimer = null }
  for (const t of saveTimers.values()) clearTimeout(t)
  saveTimers.clear()
  // Reset local UI state — pool selection back to the novel library.
  activePool.value = 'novel'
  loading.value = true
  try {
    await refresh()
    if (chapter.value?.novel_id) {
      store.loadAssets(chapter.value.novel_id).catch(() => {})
    }
    schedulePoll()
  } finally {
    loading.value = false
  }
}
onUnmounted(() => {
  if (pollTimer) clearTimeout(pollTimer)
  for (const t of saveTimers.values()) clearTimeout(t)
  saveTimers.clear()
})

async function refresh({ silent = false } = {}) {
  const prevAssetsStatus = chapter.value?.assets_status || 'none'
  if (!silent) loading.value = true
  try {
    await store.openChapter(route.params.cid)
    const novelId = chapter.value?.novel_id
    if (novelId) {
      // Reload assets if they just became ready, or on first load when already ready
      if (prevAssetsStatus === 'generating' && chapter.value?.assets_status === 'ready') {
        store.loadAssets(novelId).catch(() => {})
      } else if (!store.assets && chapter.value?.assets_status === 'ready') {
        store.loadAssets(novelId).catch(() => {})
      }
    }
    // Keep public pool fresh too, but only if already loaded
    if (store.publicAssets) {
      store.loadPublicAssets().catch(() => {})
    }
  } catch (e) {
    if (!silent) message.error('加载失败')
  } finally {
    if (!silent) loading.value = false
  }
}

function shouldPoll() {
  if (analysisStatus.value === 'analyzing') return true
  if (plots.value.some(p => p.storyboard_status === 'extracting')) return true
  if (assetsStatus.value === 'generating') return true
  return false
}

function schedulePoll() {
  if (pollTimer) clearTimeout(pollTimer)
  if (shouldPoll()) {
    pollTimer = setTimeout(async () => {
      await refresh({ silent: true })
      schedulePoll()
    }, 3000)
  }
}

async function onAnalyze() {
  try {
    message.info('正在解析本章，约 15 秒，请稍候…')
    await store.analyzeChapter(route.params.cid)
    schedulePoll()
  } catch (e) {
    message.error(e.response?.data?.error || '解析失败')
  }
}

async function onStoryboard(plot) {
  try {
    message.info('正在生成分镜，约 12 秒，请稍候…')
    await store.storyboardForPlot(plot.id)
    schedulePoll()
  } catch (e) {
    message.error(e.response?.data?.error || '生成失败')
  }
}

async function onGenerateAssets() {
  try {
    message.info('正在生成人物/场景图，每张约 60 秒，请稍候…')
    await store.generateAssets(route.params.cid)
    schedulePoll()
  } catch (e) {
    message.error(e.response?.data?.error || '生成失败')
  }
}

function scheduleSave(shot) {
  let t = saveTimers.get(shot.id)
  if (t) clearTimeout(t)
  t = setTimeout(() => {
    saveTimers.delete(shot.id)
    store.saveShot(shot.id, {
      scene: shot.scene,
      characters: shot.characters,
      prompt: shot.prompt,
      dialogue: shot.dialogue,
      camera: shot.camera,
    }).catch(() => message.error('保存失败'))
  }, 500)
  saveTimers.set(shot.id, t)
}

function statusText(s) {
  return { none: '未解析', analyzing: '解析中', ready: '已就绪', failed: '失败' }[s] || s
}
function storyboardStatusText(s) {
  return { none: '未生成', extracting: '生成中', ready: '已就绪', failed: '失败' }[s] || s
}
</script>

<style scoped>
.chapter-storyboard-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.storyboard-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--color-tint-white-06);
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
  flex: 1;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  border: 1px solid var(--color-tint-white-08);
  background: transparent;
  border-radius: 8px;
  color: var(--color-text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
  font-family: inherit;
  flex-shrink: 0;
}
.back-btn:hover {
  border-color: rgba(0, 202, 224, 0.35);
  color: #00cae0;
}

.topbar-title {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  overflow: hidden;
}

.topbar-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 999px;
  border: 1px solid transparent;
  white-space: nowrap;
}
.status-badge.status-none {
  color: var(--color-text-muted);
  border-color: var(--color-tint-white-12);
  background: var(--color-tint-white-04);
}
.status-badge.status-analyzing,
.status-badge.status-extracting {
  color: #ffd28f;
  border-color: rgba(255, 184, 92, 0.4);
  background: rgba(255, 184, 92, 0.14);
}
.status-badge.status-ready {
  color: #8cefff;
  border-color: rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.12);
}
.status-badge.status-failed {
  color: #ff9d9d;
  border-color: rgba(239, 68, 68, 0.45);
  background: rgba(239, 68, 68, 0.16);
}

.storyboard-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

/* Analyze prompt */
.analyze-block {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 22px 24px;
  background: var(--color-tint-white-02);
  border: 1px solid rgba(0, 202, 224, 0.28);
  border-left: 3px solid #00cae0;
  border-radius: 14px;
  box-shadow: 0 2px 12px var(--color-tint-black-30);
}
.analyze-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgba(0, 202, 224, 0.14);
  border: 1px solid rgba(0, 202, 224, 0.25);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #00cae0;
}
.analyze-icon svg {
  width: 22px;
  height: 22px;
}
.analyze-info {
  flex: 1;
  min-width: 0;
}
.analyze-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.analyze-desc {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

/* Generate button */
.generate-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
  flex-shrink: 0;
  border-radius: 10px;
  border: 1px solid rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.16);
  color: #d8fbff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.generate-btn.small {
  height: 28px;
  padding: 0 12px;
  font-size: 12px;
}
.generate-btn:hover:not(:disabled) {
  background: rgba(0, 202, 224, 0.28);
  box-shadow: 0 2px 10px rgba(0, 202, 224, 0.2);
}
.generate-btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}
.generate-btn.busy {
  background: rgba(0, 202, 224, 0.1);
}
.btn-icon {
  width: 15px;
  height: 15px;
}
.generate-btn.small .btn-icon {
  width: 13px;
  height: 13px;
}
.btn-icon.spin {
  animation: shot-spin 0.9s linear infinite;
}
@keyframes shot-spin {
  to { transform: rotate(360deg); }
}

/* Analysis cards */
.analysis-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
  margin-bottom: 18px;
}
.info-card {
  background: var(--color-tint-white-02);
  border: 1px solid rgba(0, 202, 224, 0.22);
  border-left: 3px solid #00cae0;
  border-radius: 14px;
  padding: 14px 18px;
  box-shadow: 0 2px 12px var(--color-tint-black-30);
}
.info-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.info-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #00cae0;
  box-shadow: 0 0 8px rgba(0, 202, 224, 0.6);
}
.info-title {
  font-size: 13px;
  font-weight: 700;
  color: #00cae0;
  letter-spacing: 0.04em;
}
.info-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

/* Assets section */
.assets-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px 18px;
  margin-bottom: 18px;
  background: var(--color-tint-white-02);
  border: 1px solid rgba(0, 202, 224, 0.22);
  border-left: 3px solid #00cae0;
  border-radius: 14px;
  box-shadow: 0 2px 12px var(--color-tint-black-30);
}
.assets-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.assets-title-wrap {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.assets-title-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #00cae0;
  box-shadow: 0 0 8px rgba(0, 202, 224, 0.6);
  flex-shrink: 0;
}
.assets-title {
  font-size: 14px;
  font-weight: 700;
  color: #00cae0;
  letter-spacing: 0.04em;
}
.asset-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.asset-group-label {
  font-size: 12px;
  font-weight: 700;
  color: #00cae0;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  opacity: 0.9;
}
.asset-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}
.asset-card {
  background: var(--color-tint-white-03);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: border-color .2s, box-shadow .2s, transform .2s;
}
.asset-card:hover {
  border-color: rgba(0, 202, 224, 0.35);
  box-shadow: 0 6px 18px var(--color-tint-black-30);
  transform: translateY(-2px);
}
.asset-img {
  width: 100%;
  height: 150px;
  object-fit: cover;
  display: block;
  background: var(--color-tint-white-04);
  border-radius: 8px 8px 0 0;
}
.asset-img-placeholder {
  width: 100%;
  height: 150px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-tint-white-04);
  color: var(--color-text-muted);
  font-size: 12px;
}
.asset-info {
  padding: 8px 10px 10px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-height: 0;
}
.asset-name {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.asset-desc {
  margin: 0;
  font-size: 11px;
  line-height: 1.5;
  color: var(--color-text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.assets-empty {
  padding: 8px 0;
  display: flex;
  justify-content: center;
}

/* Pool toggle */
.asset-pool-toggle {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  background: var(--color-tint-white-03);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 10px;
  align-self: flex-start;
}
.pool-tab {
  height: 28px;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 600;
  border-radius: 7px;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.pool-tab:hover {
  color: var(--color-text-primary);
}
.pool-tab.active {
  background: rgba(0, 202, 224, 0.18);
  color: #d8fbff;
  border: 1px solid rgba(0, 202, 224, 0.35);
}

/* Move button on asset card */
.asset-move-btn {
  align-self: flex-start;
  margin-top: 4px;
  height: 22px;
  padding: 0 8px;
  border: 1px solid rgba(0, 202, 224, 0.3);
  background: rgba(0, 202, 224, 0.08);
  color: #8cefff;
  font-size: 11px;
  font-weight: 600;
  border-radius: 6px;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.asset-move-btn:hover {
  background: rgba(0, 202, 224, 0.22);
  border-color: rgba(0, 202, 224, 0.5);
}

/* Plot list */
.plot-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.plot-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  padding: 14px 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color .2s, box-shadow .2s;
}
.plot-card:hover {
  border-color: var(--color-tint-white-12);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}
.plot-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.plot-title-wrap {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}
.plot-index {
  font-size: 13px;
  font-weight: 700;
  color: #00cae0;
  flex-shrink: 0;
}
.plot-title-text {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.plot-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.plot-summary {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  word-break: break-word;
}

/* Shots sub-grid */
.shots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.shot-card {
  background: var(--color-tint-white-03);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: border-color .2s, box-shadow .2s;
}
.shot-card:hover {
  border-color: var(--color-tint-white-12);
  box-shadow: 0 6px 18px var(--color-tint-black-30);
}
.shot-header {
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-tint-white-06);
  background: var(--color-tint-white-03);
}
.shot-index {
  font-size: 11px;
  font-weight: 600;
  color: #00cae0;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.shot-body {
  padding: 10px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.shot-field {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.shot-field-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-muted);
  letter-spacing: 0.02em;
}

.plot-empty {
  padding: 8px 0;
}

.storyboard-empty-wrap {
  margin-top: 60px;
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .storyboard-topbar { padding: 12px 14px; }
  .storyboard-scroll { padding: 12px 14px 16px; }
  .analysis-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }
  .shots-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }
  .analyze-block {
    flex-wrap: wrap;
    padding: 16px;
  }
  .plot-header {
    flex-wrap: wrap;
  }
}
</style>
