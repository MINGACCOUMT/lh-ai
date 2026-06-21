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
  return {
    novels, currentNovel, chapters, chapterTotal, chapterLimit, currentStoryboard, chapterDetail,
    loadNovels, uploadNovel, openNovel, loadMoreChapters, removeNovel, loadStoryboard, triggerStoryboard,
    openChapter, analyzeChapter, storyboardForPlot, saveShot,
  }
})
