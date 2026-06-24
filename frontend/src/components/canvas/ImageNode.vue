<script setup>
import { ref } from 'vue'
import { Handle, Position } from '@vue-flow/core'

const props = defineProps({
  id: { type: String, required: true },
  data: { type: Object, default: () => ({}) }
})

const editing = ref(false)
const urlInput = ref(props.data.imageUrl || '')

const startEdit = () => {
  urlInput.value = props.data.imageUrl || ''
  editing.value = true
}

const applyUrl = () => {
  if (!editing.value) return
  editing.value = false
  const url = urlInput.value.trim()
  // Mutate the reactive data object so Vue Flow's toObject() picks up the change.
  props.data.imageUrl = url
}
</script>

<template>
  <div class="image-node" :class="{ selected: data.selected }" @dblclick="startEdit">
    <img v-if="data.imageUrl" :src="data.imageUrl" class="node-img" alt="" />
    <div v-else class="no-img">拖入图片 URL 或双击编辑</div>
    <input
      v-if="editing"
      ref="urlInputEl"
      v-model="urlInput"
      class="url-input"
      placeholder="图片 URL"
      @keyup.enter="applyUrl"
      @blur="applyUrl"
      @click.stop
      @dblclick.stop
    />
    <Handle type="source" :position="Position.Right" />
  </div>
</template>

<style scoped>
.image-node {
  width: 200px;
  background: var(--color-card-bg, #1e2530);
  border: 1px solid var(--color-divider, #2a3340);
  border-radius: 10px;
  padding: 6px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.28);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.image-node.selected {
  border-color: #00cae0;
  box-shadow: 0 0 0 2px rgba(0, 202, 224, 0.35), 0 4px 14px rgba(0, 0, 0, 0.28);
}
.node-img {
  width: 100%;
  height: auto;
  display: block;
  border-radius: 6px;
  background: #000;
}
.no-img {
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted, #8a93a3);
  font-size: 12px;
  text-align: center;
  padding: 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.03);
}
.url-input {
  width: 100%;
  margin-top: 6px;
  box-sizing: border-box;
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid rgba(0, 202, 224, 0.4);
  background: rgba(0, 0, 0, 0.25);
  color: var(--color-text-primary, #e6eaf0);
  font-size: 12px;
  outline: none;
}
.url-input:focus {
  border-color: #00cae0;
}
</style>
