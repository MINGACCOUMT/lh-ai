import axios from 'axios'

const instance = axios.create({ baseURL: '/api' })

export function usePrompts() {
  return {
    listPrompts: (params) => instance.get('/prompts', { params })
  }
}
