<template>
  <div class="novel-list-page">
    <!-- Header -->
    <div class="novel-header">
      <div class="novel-header-left">
        <h2 class="novel-page-title">{{ t('novel.title') }}</h2>
        <span v-if="store.novels.length" class="novel-count">{{ store.novels.length }}</span>
      </div>
      <n-upload :show-file-list="false" accept=".txt" :custom-request="handleUpload">
        <button class="upload-btn" :disabled="uploading">
          <svg v-if="uploading" class="btn-icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-6.219-8.56" />
          </svg>
          <svg v-else class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
            <polyline points="17 8 12 3 7 8" />
            <line x1="12" y1="3" x2="12" y2="15" />
          </svg>
          {{ t('novel.upload') }}
        </button>
      </n-upload>
    </div>

    <!-- Grid -->
    <div class="novel-scroll">
      <div v-if="store.novels.length" class="novel-grid">
        <article
          v-for="n in store.novels"
          :key="n.id"
          class="novel-card"
          @click="router.push({ name: 'novel-detail', params: { id: n.id } })"
        >
          <div class="novel-cover">
            <div class="novel-cover-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
              </svg>
            </div>
            <span class="novel-chapter-count">{{ n.chapter_count }}</span>
          </div>
          <div class="novel-info">
            <p class="novel-title-text">{{ n.title }}</p>
            <div class="novel-meta">
              <span class="novel-date">{{ formatDate(n.created_at) }}</span>
              <div class="novel-actions">
                <button
                  class="action-btn open-btn"
                  :title="t('novel.open')"
                  @click.stop="router.push({ name: 'novel-detail', params: { id: n.id } })"
                >
                  <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                    <path d="M14 5h5v5" />
                    <path d="M10 14L19 5" />
                    <path d="M19 14v4a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1h4" />
                  </svg>
                </button>
                <n-popconfirm @positive-click="store.removeNovel(n.id)">
                  <template #trigger>
                    <button class="action-btn delete-btn" :title="t('novel.delete')" @click.stop>
                      <svg class="action-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                        <path d="M4 7h16" />
                        <path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" />
                        <path d="M8 7v12a1 1 0 0 0 1 1h6a1 1 0 0 0 1-1V7" />
                        <path d="M10 11v6" />
                        <path d="M14 11v6" />
                      </svg>
                    </button>
                  </template>
                  {{ t('novel.deleteConfirm') }}
                </n-popconfirm>
              </div>
            </div>
          </div>
        </article>
      </div>

      <div v-else-if="!loading" class="novel-empty-wrap">
        <NEmpty :description="t('novel.empty')" />
        <n-upload :show-file-list="false" accept=".txt" :custom-request="handleUpload">
          <button class="empty-upload-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
              <polyline points="17 8 12 3 7 8" />
              <line x1="12" y1="3" x2="12" y2="15" />
            </svg>
            {{ t('novel.upload') }}
          </button>
        </n-upload>
      </div>

      <div v-if="loading" class="loading-more">{{ t('assets.loading') }}</div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NEmpty, useMessage } from 'naive-ui'
import { useNovelStore } from '../stores/novel'
import { useUserStore } from '../stores/user'
import { useLocaleStore } from '../stores/locale'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()
const store = useNovelStore()
const user = useUserStore()
const localeStore = useLocaleStore()
const uploading = ref(false)
const loading = ref(false)

onMounted(async () => {
  if (!user.isLoggedIn) { user.openAuth(); return }
  loading.value = true
  try {
    await store.loadNovels()
  } finally {
    loading.value = false
  }
})

function formatDate(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ''
  const locale = localeStore.locale === 'en' ? 'en-US' : 'zh-CN'
  return new Intl.DateTimeFormat(locale, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(d)
}

async function handleUpload({ file }) {
  uploading.value = true
  try {
    const res = await store.uploadNovel(file.file)
    if (res.warning) message.warning(res.warning)
    message.success('上传成功')
    router.push({ name: 'novel-detail', params: { id: res.novel_id } })
  } catch (e) {
    message.error(e.response?.data?.error || '上传失败')
  } finally {
    uploading.value = false
  }
}
</script>

<style scoped>
.novel-list-page {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.novel-header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--color-tint-white-06);
}

.novel-header-left {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
}

.novel-page-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.novel-count {
  font-size: 12px;
  color: var(--color-text-muted);
  background: var(--color-tint-white-04);
  padding: 1px 8px;
  border-radius: 999px;
}

.upload-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
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
.upload-btn:hover:not(:disabled) {
  background: rgba(0, 202, 224, 0.28);
  box-shadow: 0 2px 10px rgba(0, 202, 224, 0.2);
}
.upload-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
.btn-icon {
  width: 15px;
  height: 15px;
}
.btn-icon.spin {
  animation: novel-spin 0.9s linear infinite;
}
@keyframes novel-spin {
  to { transform: rotate(360deg); }
}

.novel-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

.novel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

.novel-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  transition: all .25s;
}
.novel-card:hover {
  border-color: var(--color-tint-white-12);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}

.novel-cover {
  position: relative;
  aspect-ratio: 16 / 7;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(0, 202, 224, 0.10), rgba(102, 126, 234, 0.10));
  border-bottom: 1px solid var(--color-tint-white-06);
}

.novel-cover-icon {
  width: 40px;
  height: 40px;
  color: rgba(0, 202, 224, 0.65);
}
.novel-cover-icon svg {
  width: 100%;
  height: 100%;
}

.novel-chapter-count {
  position: absolute;
  top: 8px;
  right: 8px;
  background: var(--color-actions-overlay);
  color: #fff;
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 8px;
  backdrop-filter: blur(4px);
}

.novel-info {
  padding: 12px;
}

.novel-title-text {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--color-text-primary);
  margin: 0 0 10px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 2.8em;
}

.novel-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.novel-date {
  font-size: 11px;
  color: var(--color-text-muted);
}

.novel-actions {
  display: flex;
  gap: 6px;
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
.action-btn.delete-btn:hover {
  border-color: rgba(239, 68, 68, 0.45);
  background: rgba(239, 68, 68, 0.2);
}

.loading-more {
  text-align: center;
  padding: 20px;
  color: var(--color-text-muted);
  font-size: 13px;
}

.novel-empty-wrap {
  margin-top: 80px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.empty-upload-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
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
.empty-upload-btn:hover {
  background: rgba(0, 202, 224, 0.28);
}
.empty-upload-btn svg {
  width: 14px;
  height: 14px;
}

@media (max-width: 768px) {
  .novel-header { padding: 12px 14px; }
  .novel-scroll { padding: 12px 14px 16px; }
  .novel-grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 10px;
  }
  .novel-info { padding: 10px; }
}

@media (max-width: 380px) {
  .novel-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
}
</style>
