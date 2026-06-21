<template>
  <div class="novel-detail">
    <n-page-header @back="router.push({name:'novel-list'})">
      <template #title>{{ store.currentNovel?.title || '...' }}</template>
    </n-page-header>
    <n-list style="margin-top:16px">
      <n-list-item v-for="ch in store.chapters" :key="ch.id">
        <n-thing :title="`第${ch.chapter_index}章 · ${ch.title || ''}`">
          <template #description>
            <n-tag :type="statusType(ch.storyboard_status)" size="small">{{ statusText(ch.storyboard_status) }}</n-tag>
          </template>
          <template #action>
            <n-button size="small" @click="router.push({name:'chapter-storyboard',params:{cid:ch.id}})">{{ t('novel.storyboard') }}</n-button>
          </template>
        </n-thing>
      </n-list-item>
    </n-list>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNovelStore } from '../stores/novel'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const store = useNovelStore()

onMounted(() => store.openNovel(route.params.id))

function statusType(s) { return { none: 'default', extracting: 'info', ready: 'success', failed: 'error' }[s] || 'default' }
function statusText(s) { return { none: '未抽取', extracting: '抽取中', ready: '就绪', failed: '失败' }[s] || s }
</script>
