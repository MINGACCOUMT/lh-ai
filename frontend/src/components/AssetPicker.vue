<template>
  <n-modal :show="show" @update:show="$emit('update:show', $event)" preset="card" :title="title || '从素材库选择'" style="width: 720px; max-width: 90vw;">
    <div v-if="loading" style="text-align: center; padding: 40px"><n-spin /></div>
    <div v-else-if="images.length === 0" style="text-align: center; padding: 40px; color: var(--color-text-3)">素材库还没有图片</div>
    <div v-else class="picker-grid">
      <div
        v-for="img in images" :key="img.id + '|' + img.url"
        class="picker-item"
        :class="{ selected: isSelected(img.url), disabled: isExcluded(img.url) }"
        @click="onItemClick(img.url)"
      >
        <img :src="img.url" class="picker-thumb" :alt="img.prompt || ''" />
        <div v-if="isSelected(img.url)" class="picker-check">✓</div>
        <div v-else-if="isExcluded(img.url)" class="picker-mask">已添加</div>
      </div>
    </div>
    <template #footer>
      <div style="display: flex; justify-content: space-between; align-items: center;">
        <span style="color: var(--color-text-3); font-size: 13px;">已选 {{ selected.length }} 张</span>
        <n-button type="primary" :disabled="selected.length === 0" @click="onConfirm">确认选择</n-button>
      </div>
    </template>
  </n-modal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { NModal, NButton, NSpin, useMessage } from 'naive-ui'
import axios from 'axios'

const props = defineProps({
  show: Boolean,
  mode: { type: String, default: 'multi' }, // 'multi' or 'single'
  title: String,
  // 已存在的 URL 列表（防止重复添加）
  excludeUrls: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:show', 'confirm'])

const message = useMessage()
const loading = ref(false)
const images = ref([])
const selected = ref([])

watch(() => props.show, async (val) => {
  if (val) {
    selected.value = []
    await loadImages()
  }
})

async function loadImages() {
  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const res = await axios.get('/api/generations?type=image&limit=100', {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    const items = res.data.items || res.data.generations || []
    const seen = new Set()
    const flat = []
    for (const g of items) {
      let urls = []
      try { urls = JSON.parse(g.images || '[]') } catch { /* ignore */ }
      for (const url of urls) {
        if (!url || seen.has(url)) continue
        seen.add(url)
        flat.push({ id: g.id, url, prompt: g.prompt })
      }
    }
    images.value = flat
  } catch (e) {
    message.error('加载素材库失败')
  } finally {
    loading.value = false
  }
}

function isSelected(url) { return selected.value.includes(url) }
function isExcluded(url) { return props.excludeUrls.includes(url) }

function onItemClick(url) {
  if (isExcluded(url)) return
  if (props.mode === 'single') {
    selected.value = [url]
  } else {
    if (isSelected(url)) {
      selected.value = selected.value.filter(u => u !== url)
    } else {
      selected.value.push(url)
    }
  }
}

function onConfirm() {
  emit('confirm', props.mode === 'single' ? selected.value[0] : [...selected.value])
  emit('update:show', false)
}
</script>

<style scoped>
.picker-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 10px; max-height: 420px; overflow-y: auto; padding: 4px; }
.picker-item { position: relative; cursor: pointer; border-radius: 8px; overflow: hidden; border: 2px solid transparent; aspect-ratio: 1; background: rgba(255,255,255,0.04); }
.picker-item.selected { border-color: #00cae0; }
.picker-item.disabled { cursor: not-allowed; }
.picker-thumb { width: 100%; height: 100%; object-fit: cover; }
.picker-check { position: absolute; top: 4px; right: 4px; width: 22px; height: 22px; border-radius: 50%; background: #00cae0; color: #000; display: flex; align-items: center; justify-content: center; font-size: 14px; font-weight: bold; }
.picker-mask { position: absolute; inset: 0; background: rgba(0,0,0,0.55); color: #fff; display: flex; align-items: center; justify-content: center; font-size: 12px; }
</style>
