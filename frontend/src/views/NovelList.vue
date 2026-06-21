<template>
  <div class="novel-list">
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
      <h2>{{ t('novel.title') }}</h2>
      <n-upload :show-file-list="false" accept=".txt" :custom-request="handleUpload">
        <n-button type="primary" :loading="uploading">{{ t('novel.upload') }}</n-button>
      </n-upload>
    </div>
    <n-list v-if="store.novels.length">
      <n-list-item v-for="n in store.novels" :key="n.id">
        <n-thing :title="n.title" :description="`${n.chapter_count} 章 · ${new Date(n.created_at).toLocaleString()}`">
          <template #action>
            <n-button size="small" @click="router.push({name:'novel-detail',params:{id:n.id}})">{{ t('novel.open') }}</n-button>
            <n-popconfirm @positive-click="store.removeNovel(n.id)">
              <template #trigger><n-button size="small" quaternary type="error">{{ t('novel.delete') }}</n-button></template>
              {{ t('novel.deleteConfirm') }}
            </n-popconfirm>
          </template>
        </n-thing>
      </n-list-item>
    </n-list>
    <n-empty v-else :description="t('novel.empty')" />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '../stores/novel'
import { useUserStore } from '../stores/user'

const { t } = useI18n()
const router = useRouter()
const store = useNovelStore()
const user = useUserStore()
const uploading = ref(false)

onMounted(async () => {
  if (!user.isLoggedIn) { user.openAuth(); return }
  await store.loadNovels()
})

async function handleUpload({ file }) {
  uploading.value = true
  try {
    const res = await store.uploadNovel(file.file)
    if (res.warning) window.$message?.warning(res.warning)
    window.$message?.success('上传成功')
    router.push({ name: 'novel-detail', params: { id: res.novel_id } })
  } catch (e) {
    window.$message?.error(e.response?.data?.error || '上传失败')
  } finally {
    uploading.value = false
  }
}
</script>
