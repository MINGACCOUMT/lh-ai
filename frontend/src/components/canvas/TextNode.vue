<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  id: { type: String, required: true },
  data: { type: Object, default: () => ({}) }
})

// Local editable buffer seeded from node data. We push edits back into the
// reactive data object so Vue Flow's toObject() serializes the latest text.
const text = ref(props.data.text || '')

watch(
  () => props.data.text,
  (v) => {
    if (v !== text.value) text.value = v || ''
  }
)

const updateData = () => {
  props.data.text = text.value
}
</script>

<template>
  <div class="text-node" :class="{ selected: data.selected }">
    <textarea
      v-model="text"
      class="node-textarea"
      placeholder="输入文字..."
      rows="4"
      @input="updateData"
      @click.stop
      @dblclick.stop
    />
  </div>
</template>

<style scoped>
.text-node {
  width: 220px;
  background: var(--color-card-bg, #1e2530);
  border: 1px solid var(--color-divider, #2a3340);
  border-radius: 10px;
  padding: 8px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.28);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.text-node.selected {
  border-color: #00cae0;
  box-shadow: 0 0 0 2px rgba(0, 202, 224, 0.35), 0 4px 14px rgba(0, 0, 0, 0.28);
}
.node-textarea {
  width: 100%;
  box-sizing: border-box;
  resize: vertical;
  min-height: 60px;
  padding: 6px 8px;
  border: none;
  background: transparent;
  color: var(--color-text-primary, #e6eaf0);
  font-size: 13px;
  line-height: 1.5;
  font-family: inherit;
  outline: none;
  border-radius: 6px;
}
.node-textarea::placeholder {
  color: var(--color-text-muted, #8a93a3);
}
.node-textarea:focus {
  background: rgba(0, 0, 0, 0.18);
}
</style>
