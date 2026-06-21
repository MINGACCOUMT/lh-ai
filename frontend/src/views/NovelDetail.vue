<template>
  <div class="novel-detail-page">
    <!-- Topbar -->
    <div class="detail-topbar">
      <button class="back-btn" @click="router.push({ name: 'novel-list' })">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="15 18 9 12 15 6" />
        </svg>
        <span>{{ t('novel.title') }}</span>
      </button>
      <div class="topbar-title">
        <span class="topbar-name">{{ store.currentNovel?.title || '…' }}</span>
        <span v-if="store.chapters.length" class="topbar-count">{{ store.chapters.length }}</span>
      </div>
    </div>

    <!-- Chapter list -->
    <div class="chapter-scroll">
      <div v-if="store.chapters.length" class="chapter-grid">
        <article
          v-for="ch in store.chapters"
          :key="ch.id"
          class="chapter-card"
          @click="router.push({ name: 'chapter-storyboard', params: { cid: ch.id } })"
        >
          <div class="chapter-index">
            <span class="chapter-index-num">{{ ch.chapter_index }}</span>
          </div>
          <div class="chapter-body">
            <p class="chapter-title-text">{{ ch.title || '—' }}</p>
            <div class="chapter-meta">
              <span class="status-badge" :class="`status-${statusKey(ch.storyboard_status)}`">
                {{ statusText(ch.storyboard_status) }}
              </span>
            </div>
          </div>
          <div class="chapter-action">
            <button
              class="action-btn"
              :class="{ 'action-btn-ready': ch.storyboard_status === 'ready' }"
              @click.stop="router.push({ name: 'chapter-storyboard', params: { cid: ch.id } })"
            >
              <svg v-if="ch.storyboard_status === 'ready'" class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
              <svg v-else class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M5 3v4M3 5h4M6 17v4M4 19h4" />
                <path d="M13 3l2.5 6.5L22 12l-6.5 2.5L13 21l-2.5-6.5L4 12l6.5-2.5L13 3z" />
              </svg>
              <span class="action-label">{{ ch.storyboard_status === 'ready' ? '查看分镜' : '解析' }}</span>
            </button>
          </div>
        </article>
      </div>

      <div v-if="hasMore" class="load-more-wrap">
        <button class="load-more-btn" :disabled="loadingMore" @click="onLoadMore">
          <svg v-if="loadingMore" class="btn-icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-6.219-8.56" />
          </svg>
          {{ loadingMore ? '加载中…' : `加载更多 (剩余 ${remaining})` }}
        </button>
      </div>

      <div v-if="!store.chapters.length" class="chapter-empty">
        <NEmpty :description="t('novel.empty')" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

const loadingMore = ref(false)

const hasMore = computed(() => store.chapters.length < store.chapterTotal)
const remaining = computed(() => Math.max(0, store.chapterTotal - store.chapters.length))

onMounted(() => store.openNovel(route.params.id))

async function onLoadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    await store.loadMoreChapters(route.params.id)
  } catch (e) {
    window.$message?.error('加载更多失败')
  } finally {
    loadingMore.value = false
  }
}

function statusKey(s) { return ({ none: 'none', extracting: 'extracting', ready: 'ready', failed: 'failed' })[s] || 'none' }
function statusText(s) { return { none: '未抽取', extracting: '抽取中', ready: '就绪', failed: '失败' }[s] || s }
</script>

<style scoped>
.novel-detail-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.detail-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--color-tint-white-06);
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
}
.back-btn:hover {
  border-color: rgba(0, 202, 224, 0.35);
  color: #00cae0;
}

.topbar-title {
  display: flex;
  align-items: baseline;
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

.topbar-count {
  font-size: 12px;
  color: var(--color-text-muted);
  background: var(--color-tint-white-04);
  padding: 1px 8px;
  border-radius: 999px;
  white-space: nowrap;
  flex-shrink: 0;
}

.chapter-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

.chapter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 10px;
}

.chapter-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 12px;
  cursor: pointer;
  transition: all .25s;
}
.chapter-card:hover {
  border-color: var(--color-tint-white-12);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}

.chapter-index {
  flex-shrink: 0;
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background: linear-gradient(135deg, rgba(0, 202, 224, 0.12), rgba(102, 126, 234, 0.12));
  border: 1px solid rgba(0, 202, 224, 0.2);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0;
  line-height: 1;
}
.chapter-index-num {
  font-size: 15px;
  font-weight: 700;
  color: #00cae0;
}
.chapter-index-label {
  font-size: 9px;
  color: var(--color-text-muted);
  margin-top: 2px;
}

.chapter-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.chapter-title-text {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chapter-meta {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  border: 1px solid transparent;
}
.status-badge.status-none {
  color: var(--color-text-muted);
  border-color: var(--color-tint-white-12);
  background: var(--color-tint-white-04);
}
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

.chapter-action {
  flex-shrink: 0;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--color-tint-white-12);
  background: var(--color-tint-white-04);
  color: var(--color-text-secondary);
  border-radius: 8px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  font-family: inherit;
  transition: all .2s;
  white-space: nowrap;
}
.action-btn.action-btn-ready {
  border-color: rgba(0, 202, 224, 0.4);
  background: rgba(0, 202, 224, 0.16);
  color: #d8fbff;
}
.action-icon {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
  fill: none;
  flex-shrink: 0;
}
.action-btn:hover {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.2);
  color: #d8fbff;
}

.load-more-wrap {
  display: flex;
  justify-content: center;
  margin-top: 20px;
  padding-bottom: 8px;
}
.load-more-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 18px;
  border-radius: 10px;
  border: 1px solid rgba(0, 202, 224, 0.3);
  background: rgba(0, 202, 224, 0.1);
  color: #8cefff;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all .2s;
  font-family: inherit;
}
.load-more-btn:hover:not(:disabled) {
  background: rgba(0, 202, 224, 0.22);
  border-color: rgba(0, 202, 224, 0.5);
  box-shadow: 0 2px 10px rgba(0, 202, 224, 0.15);
}
.load-more-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.load-more-btn .btn-icon {
  width: 14px;
  height: 14px;
  fill: none;
}
.load-more-btn .btn-icon.spin {
  animation: load-more-spin 0.9s linear infinite;
}
@keyframes load-more-spin {
  to { transform: rotate(360deg); }
}

.chapter-empty {
  margin-top: 80px;
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .detail-topbar { padding: 12px 14px; gap: 10px; }
  .chapter-scroll { padding: 12px 14px 16px; }
  .chapter-grid {
    grid-template-columns: 1fr;
    gap: 8px;
  }
}
</style>
