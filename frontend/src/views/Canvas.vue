<script setup>
import { ref, computed, markRaw, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, NSelect } from 'naive-ui'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

import ImageNode from '../components/canvas/ImageNode.vue'
import TextNode from '../components/canvas/TextNode.vue'
import AssetPicker from '../components/AssetPicker.vue'
import { useCanvasStore } from '../stores/canvas'
import { useUserStore } from '../stores/user'
import { useModelsStore } from '../stores/models'
import { useGenerate } from '../composables/useGenerate'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const store = useCanvasStore()
const userStore = useUserStore()
const modelsStore = useModelsStore()
const { generate, pollTask } = useGenerate()

// ---- Image generation panel ----
const showGenPanel = ref(true) // generation panel visible by default
const prompt = ref('')
const selectedModel = ref('')
const generating = ref(false)
const showRefPicker = ref(false)
const refImageUrls = ref([]) // reference images for generation

const modelOptions = computed(() => {
  return (modelsStore.imageModels || []).map(m => ({
    label: m.name || m.id,
    value: m.id
  }))
})

// ---- Vue Flow state + programmatic API ----
const nodes = ref([])
const edges = ref([])

const {
  onConnect,
  addEdges,
  addNodes,
  toObject,
  screenToFlowCoordinate,
  onInit,
  fitView
} = useVueFlow()

// Register custom node types with markRaw to avoid Vue Flow's reactive-component warning.
const nodeTypes = {
  image: markRaw(ImageNode),
  text: markRaw(TextNode)
}

let vfInstance = null
onInit((instance) => {
  vfInstance = instance
})

onConnect((connection) => {
  addEdges([connection])
})

// ---- Project / persistence ----
const currentProjectId = ref(null)
const saving = ref(false)
let lastSavedSnapshot = null

const serialize = () => {
  const obj = toObject()
  return {
    name: store.currentProject?.name || '未命名画布',
    layout: JSON.stringify({
      nodes: obj.nodes || [],
      edges: obj.edges || [],
      viewport: obj.viewport || null
    })
  }
}

const save = async (silent = false) => {
  if (!currentProjectId.value) return
  const payload = serialize()
  // Skip if nothing changed since last save.
  const snapshot = JSON.stringify(payload)
  if (lastSavedSnapshot && snapshot === lastSavedSnapshot) return
  saving.value = true
  try {
    await store.saveProject(currentProjectId.value, payload)
    lastSavedSnapshot = snapshot
    if (!silent) message.success('已保存')
  } catch (e) {
    if (!silent) message.error(e.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}

// Debounced auto-save (5s) triggered by node/edge changes.
let autoSaveTimer = null
const scheduleAutoSave = () => {
  if (!currentProjectId.value) return
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(() => save(true), 5000)
}

watch([nodes, edges], () => scheduleAutoSave(), { deep: true })

// ---- Loading a project into the canvas ----
const loadLayout = (project) => {
  let layout = { nodes: [], edges: [] }
  if (project?.layout) {
    try {
      layout = typeof project.layout === 'string' ? JSON.parse(project.layout) : project.layout
    } catch {
      layout = { nodes: [], edges: [] }
    }
  }
  nodes.value = Array.isArray(layout.nodes) ? layout.nodes : []
  edges.value = Array.isArray(layout.edges) ? layout.edges : []
  lastSavedSnapshot = null // force a save baseline on next save
  nextTick(() => {
    if (vfInstance && nodes.value.length) fitView({ padding: 0.2 })
  })
}

const openProject = async (id) => {
  try {
    const data = await store.openProject(id)
    currentProjectId.value = data.id
    loadLayout(data)
    // Reflect the open project in the URL without triggering a nav reload.
    if (String(route.params.projectId) !== String(id)) {
      router.replace({ name: 'canvas-project', params: { projectId: id } })
    }
  } catch (e) {
    message.error(e.response?.data?.error || '打开画布失败')
  }
}

const onNewProject = async () => {
  if (!userStore.requireAuth()) return
  try {
    const data = await store.createProject()
    await openProject(data.id)
  } catch (e) {
    message.error(e.response?.data?.error || '创建画布失败')
  }
}

const onDeleteProject = async (id, ev) => {
  ev?.stopPropagation()
  if (!confirm('确定删除该画布?')) return
  try {
    await store.deleteProject(id)
    if (currentProjectId.value === id) {
      currentProjectId.value = null
      nodes.value = []
      edges.value = []
    }
    message.success('已删除')
  } catch (e) {
    message.error(e.response?.data?.error || '删除失败')
  }
}

// ---- Adding nodes ----
let nodeSeq = 0
const genId = (prefix) => `${prefix}_${Date.now()}_${nodeSeq++}`

const centerPosition = () => {
  // Place new nodes near the viewport center. Fall back to a default if the
  // Vue Flow instance isn't ready yet (e.g. before first interaction).
  const el = document.querySelector('.vue-flow-canvas')
  if (el && vfInstance) {
    const rect = el.getBoundingClientRect()
    try {
      return screenToFlowCoordinate({ x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 })
    } catch {
      // fall through
    }
  }
  return { x: 120 + Math.random() * 80, y: 120 + Math.random() * 80 }
}

const addImageNode = () => {
  if (!ensureProject()) return
  const pos = centerPosition()
  addNodes([
    {
      id: genId('img'),
      type: 'image',
      position: pos,
      data: { imageUrl: '', label: '图片' }
    }
  ])
}

const addTextNode = () => {
  if (!ensureProject()) return
  const pos = centerPosition()
  addNodes([
    {
      id: genId('txt'),
      type: 'text',
      position: pos,
      data: { text: '', label: '文字' }
    }
  ])
}

// Create a fresh project on demand if the user starts editing without one open.
const ensureProject = async () => {
  if (currentProjectId.value) return true
  if (!userStore.requireAuth()) return false
  try {
    const data = await store.createProject()
    await openProject(data.id)
    return true
  } catch (e) {
    message.error(e.response?.data?.error || '创建画布失败')
    return false
  }
}

const sidebarCollapsed = ref(false)
const toggleSidebar = () => { sidebarCollapsed.value = !sidebarCollapsed.value }

// ---- Image generation ----
// pollTask in useGenerate is callback-based and resolves with no value, so we
// wrap it to obtain a promise that resolves with the final update.
const pollForResult = (taskId) =>
  new Promise((resolve, reject) => {
    pollTask(taskId, (update) => {
      if (update.status === 'success' || update.status === 'failed') {
        resolve(update)
      }
    })
  })

const onGenerate = async () => {
  if (!prompt.value.trim() || !selectedModel.value) {
    message.warning('请输入提示词并选择模型')
    return
  }
  if (!userStore.requireAuth()) return
  if (!(await ensureProject())) return

  generating.value = true
  message.info('正在生成图片...')

  try {
    const payload = {
      type: 'image',
      prompt: prompt.value,
      model: selectedModel.value,
      params: { aspectRatio: '1:1', imageSize: '1K' }
    }
    if (refImageUrls.value.length > 0) {
      payload.images = refImageUrls.value
    }

    const { task_id } = await generate('image', payload)
    const result = await pollForResult(task_id)

    if (result.status === 'success' && result.images?.length) {
      // Add the generated image as a new node on the canvas, offset below the
      // viewport center so it doesn't land directly on top of existing nodes.
      const pos = centerPosition()
      pos.y += 50
      addNodes([{
        id: genId('gen'),
        type: 'image',
        position: pos,
        data: {
          imageUrl: result.images[0],
          label: '生成结果',
          prompt: prompt.value,
          model: selectedModel.value
        }
      }])
      message.success('图片已生成并添加到画布')
      prompt.value = '' // clear input
    } else {
      message.error('生成失败: ' + (result.error_msg || '未知错误'))
    }

    // Refresh user credits after a successful spend.
    userStore.fetchUserInfo()
  } catch (e) {
    message.error(e.response?.data?.error || e.message || '生成失败')
  } finally {
    generating.value = false
  }
}

// ---- Lifecycle ----
onMounted(async () => {
  if (!userStore.requireAuth()) return
  try {
    await store.loadProjects()
  } catch (e) {
    // Non-fatal: sidebar just shows empty.
  }
  // Load image models for the generation panel; default-select the first one.
  modelsStore.loadModels().then(() => {
    if (modelOptions.value.length && !selectedModel.value) {
      selectedModel.value = modelOptions.value[0].value
    }
  })
  const pid = route.params.projectId
  if (pid) await openProject(pid)
})

onBeforeUnmount(() => {
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  // Best-effort flush of pending edits when leaving the page.
  if (currentProjectId.value) save(true)
})
</script>

<template>
  <div class="canvas-page">
    <!-- Sidebar -->
    <aside class="canvas-sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header">
        <span v-if="!sidebarCollapsed" class="sidebar-title">画布</span>
        <button class="collapse-btn" @click="toggleSidebar" :aria-label="sidebarCollapsed ? '展开' : '收起'">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline v-if="sidebarCollapsed" points="9 18 15 12 9 6" />
            <polyline v-else points="15 18 9 12 15 6" />
          </svg>
        </button>
      </div>

      <button class="new-project-btn" @click="onNewProject">
        <span class="plus">+</span>
        <span v-if="!sidebarCollapsed">新建画布</span>
      </button>

      <div class="project-list" v-if="!sidebarCollapsed">
        <button
          v-for="p in store.projects"
          :key="p.id"
          class="project-item"
          :class="{ active: currentProjectId === p.id }"
          @click="openProject(p.id)"
        >
          <span class="project-name">{{ p.name || '未命名画布' }}</span>
          <span class="project-del" @click="(ev) => onDeleteProject(p.id, ev)" aria-label="删除">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="3 6 5 6 21 6" />
              <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
              <path d="M10 11v6M14 11v6" />
            </svg>
          </span>
        </button>
        <div v-if="!store.projects.length" class="empty-hint">还没有画布</div>
      </div>
    </aside>

    <!-- Main canvas -->
    <div class="canvas-main">
      <div class="canvas-toolbar">
        <button class="tool-btn" @click="addImageNode">🖼 添加图片</button>
        <button class="tool-btn" @click="addTextNode">📝 添加文字</button>
        <button class="tool-btn" :class="{ active: showGenPanel }" @click="showGenPanel = !showGenPanel">🎨 生图</button>
        <button class="tool-btn primary" :disabled="saving || !currentProjectId" @click="() => save(false)">
          <span v-if="saving">保存中…</span>
          <span v-else>💾 保存</span>
        </button>
        <div class="toolbar-spacer"></div>
        <div class="project-tag" v-if="store.currentProject?.name">
          {{ store.currentProject.name }}
        </div>
      </div>

      <!-- Generation Panel -->
      <div v-if="showGenPanel && currentProjectId" class="gen-panel">
        <textarea v-model="prompt" class="gen-input" placeholder="输入提示词生成图片..." rows="2"></textarea>
        <div class="gen-controls">
          <n-select v-model:value="selectedModel" :options="modelOptions" size="small" style="width: 160px" placeholder="选择模型" />
          <button class="gen-ref-btn" @click="showRefPicker = true" title="参考图">
            📎 {{ refImageUrls.length }}
          </button>
          <button class="gen-btn" :disabled="!prompt.trim() || generating" @click="onGenerate">
            {{ generating ? '生成中…' : '🎨 生成' }}
          </button>
        </div>
        <!-- Reference thumbnails -->
        <div v-if="refImageUrls.length" class="ref-thumbs">
          <div v-for="(url, i) in refImageUrls" :key="i" class="ref-thumb">
            <img :src="url" />
            <button class="ref-remove" @click="refImageUrls.splice(i, 1)">×</button>
          </div>
        </div>
      </div>
      <AssetPicker v-model:show="showRefPicker" mode="multi" :exclude-urls="refImageUrls"
        @confirm="(urls) => { refImageUrls.push(...urls.filter(u => !refImageUrls.includes(u))); refImageUrls.splice(3) }" />

      <VueFlow
        v-model:nodes="nodes"
        v-model:edges="edges"
        :node-types="nodeTypes"
        class="vue-flow-canvas"
        :min-zoom="0.2"
        :max-zoom="4"
        :default-viewport="{ x: 0, y: 0, zoom: 1 }"
      >
        <Background :gap="20" :size="1" pattern-color="#2a3340" />
        <Controls />
        <MiniMap pannable zoomable />
      </VueFlow>

      <div v-if="!currentProjectId" class="canvas-empty-overlay">
        <div class="empty-card">
          <div class="empty-icon">🗺️</div>
          <div class="empty-title">选择或新建一个画布开始</div>
          <button class="tool-btn primary" @click="onNewProject">+ 新建画布</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.canvas-page {
  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  background: var(--color-bg, #0f1419);
  color: var(--color-text-primary, #e6eaf0);
  overflow: hidden;
}

/* ===== Sidebar ===== */
.canvas-sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--color-sidebar-bg, #161b22);
  border-right: 1px solid var(--color-divider, #2a3340);
  display: flex;
  flex-direction: column;
  padding: 12px 10px;
  gap: 10px;
  transition: width 0.2s ease;
  z-index: 5;
}
.canvas-sidebar.collapsed {
  width: 52px;
}
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.sidebar-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary, #aab2c0);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.collapse-btn {
  border: none;
  background: transparent;
  color: var(--color-text-muted, #8a93a3);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.collapse-btn:hover {
  background: var(--color-sidebar-hover, rgba(255, 255, 255, 0.05));
  color: var(--color-text-primary, #e6eaf0);
}
.new-project-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 9px 10px;
  border: 1px dashed rgba(0, 202, 224, 0.5);
  background: rgba(0, 202, 224, 0.06);
  color: #00cae0;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
  white-space: nowrap;
  overflow: hidden;
}
.new-project-btn:hover {
  background: rgba(0, 202, 224, 0.12);
  border-color: #00cae0;
}
.new-project-btn .plus {
  font-size: 15px;
  line-height: 1;
}
.project-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}
.project-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary, #aab2c0);
  border-radius: 8px;
  cursor: pointer;
  text-align: left;
  font-size: 13px;
  transition: background 0.15s, color 0.15s;
}
.project-item:hover {
  background: var(--color-sidebar-hover, rgba(255, 255, 255, 0.05));
  color: var(--color-text-primary, #e6eaf0);
}
.project-item.active {
  background: rgba(0, 202, 224, 0.1);
  color: #00cae0;
}
.project-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.project-del {
  display: flex;
  opacity: 0;
  color: var(--color-text-muted, #8a93a3);
  padding: 2px;
  border-radius: 4px;
  transition: opacity 0.15s, color 0.15s, background 0.15s;
}
.project-item:hover .project-del {
  opacity: 0.7;
}
.project-del:hover {
  opacity: 1;
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}
.empty-hint {
  color: var(--color-text-muted, #8a93a3);
  font-size: 12px;
  text-align: center;
  padding: 16px 0;
}

/* ===== Main ===== */
.canvas-main {
  flex: 1;
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.canvas-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--color-sidebar-bg, #161b22);
  border-bottom: 1px solid var(--color-divider, #2a3340);
  z-index: 4;
}
.tool-btn {
  border: 1px solid var(--color-divider, #2a3340);
  background: rgba(255, 255, 255, 0.03);
  color: var(--color-text-primary, #e6eaf0);
  padding: 7px 12px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  white-space: nowrap;
}
.tool-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.07);
  border-color: rgba(0, 202, 224, 0.4);
}
.tool-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.tool-btn.primary {
  background: #00cae0;
  border-color: #00cae0;
  color: #04141a;
  font-weight: 600;
}
.tool-btn.primary:hover:not(:disabled) {
  background: #33d5e7;
  border-color: #33d5e7;
}
.tool-btn.active {
  background: rgba(0, 202, 224, 0.15);
  border-color: #00cae0;
  color: #00cae0;
}

/* ===== Generation panel ===== */
.gen-panel {
  position: absolute;
  top: 56px; /* below toolbar */
  left: 50%;
  transform: translateX(-50%);
  z-index: 10;
  width: min(460px, 90%);
  background: rgba(20, 25, 35, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 202, 224, 0.2);
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}
.gen-input {
  width: 100%;
  box-sizing: border-box;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: #e6eaf0;
  padding: 8px 12px;
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
}
.gen-input:focus { border-color: #00cae0; }
.gen-controls {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  align-items: center;
}
.gen-ref-btn {
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.1);
  color: #9aa3b2;
  border-radius: 6px;
  padding: 6px 10px;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
}
.gen-btn {
  background: #00cae0;
  color: #000;
  border: none;
  border-radius: 6px;
  padding: 6px 16px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}
.gen-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.ref-thumbs {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
}
.ref-thumb {
  position: relative;
  width: 48px;
  height: 48px;
  border-radius: 6px;
  overflow: hidden;
}
.ref-thumb img { width: 100%; height: 100%; object-fit: cover; }
.ref-remove {
  position: absolute; top: 2px; right: 2px;
  background: rgba(0,0,0,0.6); color: #fff;
  border: none; border-radius: 50%; width: 16px; height: 16px;
  font-size: 10px; cursor: pointer; line-height: 1;
}
.toolbar-spacer {
  flex: 1;
}
.project-tag {
  font-size: 12px;
  color: var(--color-text-muted, #8a93a3);
  padding: 4px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.vue-flow-canvas {
  flex: 1;
  min-height: 0;
}

/* ===== Empty overlay ===== */
.canvas-empty-overlay {
  position: absolute;
  inset: 0;
  top: 0; /* toolbar height handled by flex layout above; overlay sits over canvas only */
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: center;
}
.canvas-empty-overlay .tool-btn {
  pointer-events: auto;
}
.empty-card {
  pointer-events: auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 28px 36px;
  background: color-mix(in oklab, var(--color-sidebar-bg, #161b22) 92%, #000 8%);
  border: 1px solid var(--color-divider, #2a3340);
  border-radius: 14px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.35);
}
.empty-icon {
  font-size: 36px;
}
.empty-title {
  font-size: 14px;
  color: var(--color-text-secondary, #aab2c0);
}

/* ===== Vue Flow dark-theme overrides ===== */
:deep(.vue-flow__edge-path) {
  stroke: #4a5563;
  stroke-width: 2;
}
:deep(.vue-flow__edge.selected .vue-flow__edge-path),
:deep(.vue-flow__edge:hover .vue-flow__edge-path) {
  stroke: #00cae0;
}
:deep(.vue-flow__handle) {
  width: 10px;
  height: 10px;
  background: #00cae0;
  border: 2px solid #0f1419;
}
:deep(.vue-flow__controls) {
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35);
  border-radius: 8px;
  overflow: hidden;
}
:deep(.vue-flow__controls-button) {
  background: var(--color-sidebar-bg, #161b22);
  border-bottom: 1px solid var(--color-divider, #2a3340);
  color: var(--color-text-primary, #e6eaf0);
  fill: var(--color-text-primary, #e6eaf0);
}
:deep(.vue-flow__controls-button:hover) {
  background: rgba(0, 202, 224, 0.12);
}
:deep(.vue-flow__minimap) {
  background: var(--color-sidebar-bg, #161b22);
  border-radius: 8px;
  overflow: hidden;
}

/* ===== Mobile ===== */
@media (max-width: 768px) {
  .canvas-sidebar {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 200px;
    box-shadow: 4px 0 20px rgba(0, 0, 0, 0.3);
  }
  .canvas-sidebar.collapsed {
    transform: translateX(-100%);
    width: 200px;
  }
}
</style>
