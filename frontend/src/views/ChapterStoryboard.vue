<template>
  <div class="chapter-storyboard">
    <n-page-header @back="router.back()">
      <template #title>{{ store.currentStoryboard?.title || '分镜' }}</template>
      <template #extra>
        <n-button type="primary" :loading="status==='extracting'" :disabled="status==='extracting'" @click="onGenerate">{{ t('novel.generateStoryboard') }}</n-button>
      </template>
    </n-page-header>

    <n-spin :show="status === 'extracting'">
      <div class="shots-grid" v-if="shots.length">
        <n-card v-for="s in shots" :key="s.id" :title="`分镜 ${s.shot_index}`" size="small" style="margin:8px">
          <n-space vertical>
            <n-input v-model:value="s.scene" type="textarea" :autosize="{minRows:1}" placeholder="场景" @blur="save(s)" />
            <n-input v-model:value="s.characters" placeholder="人物" @blur="save(s)" />
            <n-input v-model:value="s.prompt" type="textarea" :autosize="{minRows:2}" placeholder="图片提示词" @blur="save(s)" />
            <n-input v-model:value="s.dialogue" type="textarea" :autosize="{minRows:1}" placeholder="对白/旁白" @blur="save(s)" />
            <n-input v-model:value="s.camera" placeholder="镜头运动" @blur="save(s)" />
          </n-space>
        </n-card>
      </div>
      <n-empty v-else-if="status!=='extracting'" :description="t('novel.noStoryboard')" />
    </n-spin>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

const status = ref('none')
let timer = null

const shots = computed(() => store.currentStoryboard?.shots || [])

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
</script>

<style scoped>
.shots-grid { display:grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); margin-top:16px }
</style>
