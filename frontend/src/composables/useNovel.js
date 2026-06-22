import axios from 'axios'

const instance = axios.create({ baseURL: '/api/novel' })
instance.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export function useNovel() {
  const authHeaders = () => {
    const token = localStorage.getItem('token')
    return token ? { Authorization: `Bearer ${token}` } : {}
  }
  return {
    uploadNovel: (file) => {
      const fd = new FormData()
      fd.append('file', file)
      return instance.post('/upload', fd, { headers: { ...authHeaders(), 'Content-Type': 'multipart/form-data' }, timeout: 60000 })
    },
    listNovels: (params) => instance.get('', { params }),
    getNovel: (id, params) => instance.get(`/${id}`, { params }),
    deleteNovel: (id) => instance.delete(`/${id}`),
    analyzeChapter: (id) => instance.post(`/chapters/${id}/analyze`),
    getChapterDetail: (id) => instance.get(`/chapters/${id}`),
    storyboardForPlot: (plotId) => instance.post(`/plots/${plotId}/storyboard`),
    triggerStoryboard: (chapterId) => instance.post(`/chapters/${chapterId}/storyboard`),
    getStoryboard: (chapterId) => instance.get(`/chapters/${chapterId}/storyboard`),
    updateShot: (shotId, payload) => instance.put(`/shots/${shotId}`, payload),
    generateAssets: (chapterId) => instance.post(`/chapters/${chapterId}/assets`),
    getAssets: (novelId) => instance.get(`/${novelId}/assets`),
  }
}
