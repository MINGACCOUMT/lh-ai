<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { NEmpty, NSpin, useMessage } from 'naive-ui'
import { usePrompts } from '../composables/usePrompts'
import { useInspiration } from '../composables/useInspiration'
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const message = useMessage()
const userStore = useUserStore()
const { listPrompts } = usePrompts()
const { publishInspiration } = useInspiration()

const PAGE_SIZE = 24

const keyword = ref('')
const typeFilter = ref('all') // all | image | video

const items = ref([])
const total = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = computed(() => items.value.length < total.value)

let searchTimer = null

const fetchPrompts = async (reset = false) => {
  if (reset) {
    loading.value = true
  } else {
    if (!hasMore.value || loadingMore.value || loading.value) return
    loadingMore.value = true
  }
  try {
    const offset = reset ? 0 : items.value.length
    const params = { limit: PAGE_SIZE, offset }
    if (typeFilter.value !== 'all') params.type = typeFilter.value
    const kw = keyword.value.trim()
    if (kw) params.q = kw

    const { data } = await listPrompts(params)
    const list = data.items || []
    total.value = data.total || 0
    if (reset) items.value = list
    else items.value.push(...list)
  } catch (e) {
    message.error(e.response?.data?.error || t('prompts.loadFailed'))
    if (reset) items.value = []
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

const reload = () => fetchPrompts(true)

const onSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(reload, 300)
}

const setType = (type) => {
  if (typeFilter.value === type) return
  typeFilter.value = type
  reload()
}

const truncate = (text, n = 40) => {
  if (!text) return ''
  return text.length > n ? text.slice(0, n) + '…' : text
}

const isVideo = (item) => (item.mediaType || '').toLowerCase() === 'video'

const copyPrompt = async (item) => {
  try {
    await navigator.clipboard.writeText(item.prompt || '')
    message.success(t('prompts.copied'))
  } catch {
    // Fallback for environments without clipboard API
    const ta = document.createElement('textarea')
    ta.value = item.prompt || ''
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      message.success(t('prompts.copied'))
    } catch {
      message.error(t('prompts.copyFailed'))
    }
    document.body.removeChild(ta)
  }
}

const shareLoadingId = ref(null)
const shareToInspiration = async (item) => {
  if (!userStore.requireAuth()) return
  if (shareLoadingId.value === item.id) return
  shareLoadingId.value = item.id
  try {
    const post = await publishInspiration({
      source_type: 'upload',
      title: item.title || truncate(item.prompt, 30),
      description: '',
      prompt: item.prompt || '',
      tags: [],
      images: item.image ? [item.image] : [],
      video_url: '',
      cover_url: isVideo(item) && item.image ? item.image : '',
      type: isVideo(item) ? 'video' : 'image'
    })
    const reviewStatus = (post?.review_status || '').toLowerCase()
    if (reviewStatus === 'approved' || reviewStatus === '') {
      message.success(t('prompts.shareSuccess'))
    } else {
      message.success(t('prompts.shareSubmitted'))
    }
  } catch (e) {
    message.error(e.response?.data?.error || t('prompts.shareFailed'))
  } finally {
    shareLoadingId.value = null
  }
}

const expandedId = ref(null)
const toggleExpand = (item) => {
  expandedId.value = expandedId.value === item.id ? null : item.id
}

const scrollEl = ref(null)
const onScroll = (e) => {
  const el = e.target
  if (el.scrollHeight - el.scrollTop - el.clientHeight < 240) {
    fetchPrompts(false)
  }
}

onMounted(() => {
  reload()
  // Attach scroll listener once element is mounted
  setTimeout(() => {
    if (scrollEl.value && !scrollEl.value.__promptsBound) {
      scrollEl.value.addEventListener('scroll', onScroll, { passive: true })
      scrollEl.value.__promptsBound = true
    }
  }, 0)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
  if (scrollEl.value) {
    scrollEl.value.removeEventListener('scroll', onScroll)
  }
})
</script>

<template>
  <div class="prompts-page">
    <!-- Search + filter bar -->
    <div class="filter-bar">
      <div class="search-row">
        <div class="search-box">
          <svg class="search-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <path d="M21 21l-4.35-4.35" />
          </svg>
          <input
            v-model="keyword"
            type="text"
            class="search-input"
            :placeholder="t('prompts.searchPlaceholder')"
            @input="onSearchInput"
          />
        </div>
      </div>
      <div class="filter-row">
        <button :class="['filter-tab', { active: typeFilter === 'all' }]" @click="setType('all')">{{ t('prompts.typeAll') }}</button>
        <button :class="['filter-tab', { active: typeFilter === 'image' }]" @click="setType('image')">{{ t('prompts.typeImage') }}</button>
        <button :class="['filter-tab', { active: typeFilter === 'video' }]" @click="setType('video')">{{ t('prompts.typeVideo') }}</button>
      </div>
    </div>

    <!-- Grid -->
    <div ref="scrollEl" class="prompts-scroll">
      <div v-if="loading" class="state-wrap">
        <NSpin size="medium" />
      </div>
      <template v-else>
        <div v-if="items.length" class="prompts-grid">
          <article v-for="item in items" :key="item.id" class="prompt-card">
            <div class="prompt-preview" @click="toggleExpand(item)">
              <img :src="item.image" class="prompt-thumb" loading="lazy" :alt="item.title" />
              <div v-if="isVideo(item)" class="prompt-badge">视频</div>
              <div v-else class="prompt-badge image-badge">图片</div>
            </div>
            <div class="prompt-body">
              <h3 class="prompt-title" :title="item.title">{{ truncate(item.title, 28) }}</h3>
              <p
                class="prompt-text"
                :class="{ expanded: expandedId === item.id }"
              >{{ item.prompt }}</p>
              <div class="prompt-actions">
                <button class="action-btn" @click="copyPrompt(item)" :title="t('prompts.copyAction')">
                  <span class="action-emoji">📋</span>
                  <span class="action-label">{{ t('prompts.copyAction') }}</span>
                </button>
                <button
                  class="action-btn share-btn"
                  :disabled="shareLoadingId === item.id"
                  @click="shareToInspiration(item)"
                  :title="t('prompts.shareAction')"
                >
                  <span class="action-emoji">{{ shareLoadingId === item.id ? '⏳' : '✨' }}</span>
                  <span class="action-label">{{ t('prompts.shareAction') }}</span>
                </button>
              </div>
            </div>
          </article>
        </div>
        <div v-else class="state-wrap">
          <NEmpty :description="t('prompts.empty')" />
        </div>
        <div v-if="loadingMore" class="loading-more">{{ t('prompts.loading') }}</div>
        <div v-else-if="items.length && !hasMore" class="loading-more">{{ t('prompts.noMore') }}</div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.prompts-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.filter-bar {
  flex-shrink: 0;
  padding: 16px 24px 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.search-row {
  display: flex;
}

.search-box {
  position: relative;
  flex: 1;
  max-width: 480px;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  color: var(--color-text-muted);
  pointer-events: none;
}

.search-input {
  width: 100%;
  height: 38px;
  padding: 0 14px 0 36px;
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-08);
  border-radius: 10px;
  color: var(--color-text-primary);
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: all .2s;
}
.search-input::placeholder { color: var(--color-text-muted); }
.search-input:focus {
  border-color: rgba(0, 202, 224, 0.45);
  background: var(--color-tint-white-04);
}

.filter-row {
  display: flex;
  gap: 4px;
}

.filter-tab {
  padding: 7px 18px;
  background: transparent;
  border: 1px solid var(--color-tint-white-08);
  border-radius: 10px;
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.filter-tab:hover {
  background: var(--color-tint-white-04);
  border-color: var(--color-tint-white-15);
}
.filter-tab.active {
  background: rgba(0, 202, 224, 0.1);
  border-color: rgba(0, 202, 224, 0.3);
  color: #00cae0;
  font-weight: 600;
}

.prompts-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

.state-wrap {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 80px 0;
}

.prompts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 14px;
}

.prompt-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  overflow: hidden;
  transition: all .25s;
  display: flex;
  flex-direction: column;
}
.prompt-card:hover {
  border-color: var(--color-tint-white-12);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}

.prompt-preview {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  cursor: pointer;
  background: var(--color-tint-white-04);
}

.prompt-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform .3s;
}
.prompt-preview:hover .prompt-thumb {
  transform: scale(1.05);
}

.prompt-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  background: rgba(0, 202, 224, 0.85);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 8px;
  backdrop-filter: blur(4px);
}
.prompt-badge.image-badge {
  background: rgba(139, 92, 246, 0.85);
}

.prompt-body {
  padding: 10px 12px 12px;
  display: flex;
  flex-direction: column;
  flex: 1;
}

.prompt-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.prompt-text {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-secondary);
  margin: 0 0 10px;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
  word-break: break-word;
}
.prompt-text.expanded {
  -webkit-line-clamp: unset;
  overflow: visible;
}

.prompt-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}

.action-btn {
  flex: 1;
  height: 30px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  background: rgba(0, 0, 0, 0.35);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 12px;
  font-family: inherit;
  transition: all .2s;
}
.action-btn:hover:not(:disabled) {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.18);
}
.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.action-btn.share-btn:hover:not(:disabled) {
  border-color: rgba(0, 202, 224, 0.5);
  background: rgba(0, 202, 224, 0.22);
  color: #d8fbff;
}
.action-emoji {
  font-size: 13px;
  line-height: 1;
}
.action-label {
  font-size: 11px;
  font-weight: 500;
}

.loading-more {
  text-align: center;
  padding: 18px;
  color: var(--color-text-muted);
  font-size: 13px;
}

@media (max-width: 768px) {
  .filter-bar { padding: 12px 14px 0; }
  .prompts-scroll { padding: 12px 14px 16px; }
  .prompts-grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }
  .prompt-body { padding: 8px 10px 10px; }
  .action-label { display: none; }
  .action-btn { gap: 0; }
}

@media (max-width: 380px) {
  .prompts-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
}
</style>
