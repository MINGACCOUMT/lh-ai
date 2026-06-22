import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useNovel } from '../composables/useNovel'

export const useNovelStore = defineStore('novel', () => {
  const api = useNovel()
  const novels = ref([])
  const currentNovel = ref(null)
  const chapters = ref([])
  const chapterTotal = ref(0)
  const chapterLimit = ref(20)
  const currentStoryboard = ref(null)
  const chapterDetail = ref(null)
  const assets = ref(null)
  const publicAssets = ref(null)

  async function loadNovels() {
    const { data } = await api.listNovels({ limit: 50 })
    novels.value = data.items
    return data
  }
  async function uploadNovel(file) {
    const { data } = await api.uploadNovel(file)
    await loadNovels()
    return data
  }
  async function openNovel(id) {
    // Clear stale state BEFORE the await so the UI doesn't show the previous
    // novel's data while the new one is loading.
    currentNovel.value = null
    chapters.value = []
    chapterTotal.value = 0
    chapterDetail.value = null
    const { data } = await api.getNovel(id, { limit: chapterLimit.value, offset: 0 })
    currentNovel.value = data.novel
    chapters.value = data.chapters || []
    chapterTotal.value = data.chapter_total || 0
    return data
  }
  async function loadMoreChapters(id) {
    const offset = chapters.value.length
    const { data } = await api.getNovel(id, { limit: chapterLimit.value, offset })
    chapters.value = chapters.value.concat(data.chapters || [])
    chapterTotal.value = data.chapter_total || chapterTotal.value
    return data
  }
  async function removeNovel(id) {
    await api.deleteNovel(id)
    await loadNovels()
  }
  async function loadStoryboard(chapterId) {
    const { data } = await api.getStoryboard(chapterId)
    currentStoryboard.value = data
    return data
  }
  async function triggerStoryboard(chapterId) {
    await api.triggerStoryboard(chapterId)
  }
  async function openChapter(id) {
    // Clear stale chapter detail BEFORE the await so the previous chapter's
    // data doesn't bleed through while the new one loads.
    chapterDetail.value = null
    const { data } = await api.getChapterDetail(id)
    chapterDetail.value = data
    return data
  }
  async function analyzeChapter(id) {
    const { data } = await api.analyzeChapter(id)
    return data
  }
  async function storyboardForPlot(plotId) {
    const { data } = await api.storyboardForPlot(plotId)
    return data
  }
  async function saveShot(shotId, payload) {
    await api.updateShot(shotId, payload)
  }
  async function generateAssets(chapterId) {
    const { data } = await api.generateAssets(chapterId)
    return data
  }
  async function loadAssets(novelId) {
    const { data } = await api.getAssets(novelId)
    assets.value = data
    return data
  }
  async function loadPublicAssets() {
    const { data } = await api.getPublicAssets()
    publicAssets.value = data
    return data
  }
  async function moveAssetScope(assetId, novelId) {
    await api.moveAssetScope(assetId, novelId)
    // Reload whichever pools are already loaded so the UI stays fresh.
    // Use the current novel id (not the move target) for the novel-scoped reload.
    const reloads = []
    const curNovelId = chapterDetail.value?.chapter?.novel_id || currentNovel.value?.id || null
    if (assets.value && curNovelId) {
      reloads.push(api.getAssets(curNovelId).then(({ data }) => { assets.value = data }))
    }
    if (publicAssets.value) {
      reloads.push(api.getPublicAssets().then(({ data }) => { publicAssets.value = data }))
    }
    await Promise.all(reloads)
  }
  return {
    novels, currentNovel, chapters, chapterTotal, chapterLimit, currentStoryboard, chapterDetail, assets, publicAssets,
    loadNovels, uploadNovel, openNovel, loadMoreChapters, removeNovel, loadStoryboard, triggerStoryboard,
    openChapter, analyzeChapter, storyboardForPlot, saveShot, generateAssets, loadAssets, loadPublicAssets, moveAssetScope,
  }
})
