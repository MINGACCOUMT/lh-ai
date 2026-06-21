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
              :title="t('novel.storyboard')"
              @click.stop="router.push({ name: 'chapter-storyboard', params: { cid: ch.id } })"
            >
              <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <path d="M14 5h5v5" />
                <path d="M10 14L19 5" />
                <path d="M19 14v4a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h4" />
              </svg>
            </button>
          </div>
        </article>
      </div>

      <div v-else class="chapter-empty">
        <NEmpty :description="t('novel.empty')" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

onMounted(() => store.openNovel(route.params.id))

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
  width: 28px;
  height: 28px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all .2s;
}
.action-icon {
  width: 14px;
  height: 14px;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
  fill: none;
}
.action-btn:hover {
  border-color: rgba(0, 202, 224, 0.45);
  background: rgba(0, 202, 224, 0.2);
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
