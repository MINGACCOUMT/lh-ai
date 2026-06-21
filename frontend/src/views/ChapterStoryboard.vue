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
          <span class="topbar-name">{{ store.currentStoryboard?.title || t('novel.storyboard') }}</span>
          <span v-if="shots.length" class="topbar-status status-badge" :class="`status-${status}`">
            {{ statusText(status) }}
          </span>
        </div>
      </div>
      <button
        class="generate-btn"
        :class="{ busy: status === 'extracting' }"
        :disabled="status === 'extracting'"
        @click="onGenerate"
      >
        <svg v-if="status === 'extracting'" class="btn-icon spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 12a9 9 0 1 1-6.219-8.56" />
        </svg>
        <svg v-else class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M5 3v4M3 5h4M6 17v4M4 19h4" />
          <path d="M13 3l2.5 6.5L22 12l-6.5 2.5L13 21l-2.5-6.5L4 12l6.5-2.5L13 3z" />
        </svg>
        {{ generateLabel }}
      </button>
    </div>

    <!-- Shots -->
    <div class="shots-scroll">
      <section v-if="status === 'ready' && store.currentStoryboard?.outline" class="outline-card">
        <header class="outline-header">
          <span class="outline-dot"></span>
          <span class="outline-title">本章大纲</span>
        </header>
        <p class="outline-text">{{ store.currentStoryboard.outline }}</p>
      </section>

      <n-spin :show="status === 'extracting' && !shots.length">
        <div class="shots-grid" v-if="shots.length">
          <article v-for="s in shots" :key="s.id" class="shot-card">
            <header class="shot-header">
              <span class="shot-index">{{ t('novel.storyboard') }} {{ s.shot_index }}</span>
            </header>
            <div class="shot-body">
              <label class="shot-field">
                <span class="shot-field-label">场景</span>
                <n-input v-model:value="s.scene" type="textarea" :autosize="{ minRows: 1 }" placeholder="场景" @blur="save(s)" />
              </label>
              <label class="shot-field">
                <span class="shot-field-label">人物</span>
                <n-input v-model:value="s.characters" placeholder="人物" @blur="save(s)" />
              </label>
              <label class="shot-field">
                <span class="shot-field-label">图片提示词</span>
                <n-input v-model:value="s.prompt" type="textarea" :autosize="{ minRows: 2 }" placeholder="图片提示词" @blur="save(s)" />
              </label>
              <label class="shot-field">
                <span class="shot-field-label">对白 / 旁白</span>
                <n-input v-model:value="s.dialogue" type="textarea" :autosize="{ minRows: 1 }" placeholder="对白/旁白" @blur="save(s)" />
              </label>
              <label class="shot-field">
                <span class="shot-field-label">镜头运动</span>
                <n-input v-model:value="s.camera" placeholder="镜头运动" @blur="save(s)" />
              </label>
            </div>
          </article>
        </div>
        <div v-else-if="status === 'extracting'" class="shots-placeholder">{{ statusText(status) }}…</div>
        <div v-else class="shots-empty-wrap">
          <NEmpty :description="t('novel.noStoryboard')" />
        </div>
      </n-spin>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { NEmpty } from 'naive-ui'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

const status = ref('none')
let timer = null

const shots = computed(() => store.currentStoryboard?.shots || [])
const generateLabel = computed(() => {
  if (status.value === 'extracting') return '解析中…'
  if (status.value === 'ready') return '重新解析'
  return '解析本章'
})

onMounted(async () => { await poll() })
onUnmounted(() => { if (timer) clearTimeout(timer) })

async function poll() {
  try {
    const data = await store.loadStoryboard(route.params.cid)
    status.value = data.status
    if (status.value === 'extracting') {
      timer = setTimeout(poll, 3000)
    }
  } catch (e) { window.$message?.error('加载失败') }
}

async function onGenerate() {
  try {
    await store.triggerStoryboard(route.params.cid)
    status.value = 'extracting'
    store.currentStoryboard = { ...store.currentStoryboard, shots: [] }
    timer = setTimeout(poll, 3000)
  } catch (e) {
    window.$message?.error(e.response?.data?.error || '抽取失败')
  }
}

let saveTimer = null
function save(s) {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(() => {
    store.saveShot(s.id, { scene: s.scene, characters: s.characters, prompt: s.prompt, dialogue: s.dialogue, camera: s.camera })
      .catch(() => window.$message?.error('保存失败'))
  }, 400)
}

function statusText(s) {
  return { none: '未抽取', extracting: '抽取中', ready: '就绪', failed: '失败' }[s] || s
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
.btn-icon.spin {
  animation: shot-spin 0.9s linear infinite;
}
@keyframes shot-spin {
  to { transform: rotate(360deg); }
}

/* status badge reused */
.topbar-status.status-badge {
  height: 22px;
  padding: 0 8px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 999px;
  border: 1px solid transparent;
  white-space: nowrap;
}
.topbar-status.status-none {
  color: var(--color-text-muted);
  border-color: var(--color-tint-white-12);
  background: var(--color-tint-white-04);
}
.topbar-status.status-extracting {
  color: #ffd28f;
  border-color: rgba(255, 184, 92, 0.4);
  background: rgba(255, 184, 92, 0.14);
}
.topbar-status.status-ready {
  color: #8cefff;
  border-color: rgba(0, 202, 224, 0.35);
  background: rgba(0, 202, 224, 0.12);
}
.topbar-status.status-failed {
  color: #ff9d9d;
  border-color: rgba(239, 68, 68, 0.45);
  background: rgba(239, 68, 68, 0.16);
}

.shots-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px 24px;
}

.shots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
}

.outline-card {
  background: var(--color-tint-white-02);
  border: 1px solid rgba(0, 202, 224, 0.28);
  border-left: 3px solid #00cae0;
  border-radius: 14px;
  padding: 14px 18px;
  margin-bottom: 16px;
  box-shadow: 0 2px 12px var(--color-tint-black-30);
}
.outline-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.outline-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #00cae0;
  box-shadow: 0 0 8px rgba(0, 202, 224, 0.6);
}
.outline-title {
  font-size: 13px;
  font-weight: 700;
  color: #00cae0;
  letter-spacing: 0.04em;
}
.outline-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.shot-card {
  background: var(--color-tint-white-02);
  border: 1px solid var(--color-tint-white-06);
  border-radius: 14px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  transition: border-color .2s, box-shadow .2s;
}
.shot-card:hover {
  border-color: var(--color-tint-white-12);
  box-shadow: 0 8px 24px var(--color-tint-black-30);
}

.shot-header {
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-tint-white-06);
  background: var(--color-tint-white-03);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.shot-index {
  font-size: 12px;
  font-weight: 600;
  color: #00cae0;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.shot-body {
  padding: 12px 14px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.shot-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.shot-field-label {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-muted);
  letter-spacing: 0.02em;
}

.shots-placeholder {
  text-align: center;
  padding: 60px 0;
  color: var(--color-text-muted);
  font-size: 13px;
}

.shots-empty-wrap {
  margin-top: 80px;
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .storyboard-topbar { padding: 12px 14px; }
  .shots-scroll { padding: 12px 14px 16px; }
  .shots-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }
}
</style>
