import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useNovel } from '../composables/useNovel'

export const useNovelStore = defineStore('novel', () => {
  const api = useNovel()
  const novels = ref([])
  const currentNovel = ref(null)
  const chapters = ref([])
  const currentStoryboard = ref(null) // {status, shots, title, content}

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
    const { data } = await api.getNovel(id)
    currentNovel.value = data.novel
    chapters.value = data.chapters
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
  async function saveShot(shotId, payload) {
    await api.updateShot(shotId, payload)
  }
  return {
    novels, currentNovel, chapters, currentStoryboard,
    loadNovels, uploadNovel, openNovel, removeNovel, loadStoryboard, triggerStoryboard, saveShot,
  }
})
