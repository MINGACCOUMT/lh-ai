import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

const instance = axios.create({ baseURL: '/api/canvas' })
instance.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

export const useCanvasStore = defineStore('canvas', () => {
  const projects = ref([])
  const currentProject = ref(null)

  async function loadProjects() {
    const { data } = await instance.get('')
    projects.value = data.items
    return data
  }
  async function createProject(name) {
    const { data } = await instance.post('', { name: name || '未命名画布', layout: '{"nodes":[],"edges":[]}' })
    await loadProjects()
    return data
  }
  async function openProject(id) {
    const { data } = await instance.get(`/${id}`)
    currentProject.value = data
    return data
  }
  async function saveProject(id, payload) {
    await instance.put(`/${id}`, payload)
  }
  async function deleteProject(id) {
    await instance.delete(`/${id}`)
    await loadProjects()
  }
  return { projects, currentProject, loadProjects, createProject, openProject, saveProject, deleteProject }
})
