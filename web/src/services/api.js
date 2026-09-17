import axios from 'axios'

const baseURL = import.meta.env.VITE_API_BASE_URL || ''

const api = axios.create({
  baseURL,
  timeout: 120000,
  headers: { 'Content-Type': 'application/json' },
})

const unwrap = (res) => res.data?.data ?? res.data

export const API = {
  // Dashboard
  dashboard: () => api.get('/api/dashboard').then(unwrap),

  // Content
  listContent: (params) => api.get('/api/content', { params }).then(unwrap),
  getContent: (id) => api.get(`/api/content/${id}`).then(unwrap),
  getPreview: (id) => api.get(`/api/content/${id}/preview`).then(unwrap),
  generate: (payload) => api.post('/api/content/generate', payload).then(unwrap),
  updateContent: (id, payload) => api.put(`/api/content/${id}`, payload).then(unwrap),
  regenerate: (id, payload) => api.post(`/api/content/${id}/regenerate`, payload).then(unwrap),
  approve: (id) => api.post(`/api/content/${id}/approve`).then(unwrap),
  reject: (id, reason) => api.post(`/api/content/${id}/reject`, { reason }).then(unwrap),

  // Queue
  queue: () => api.get('/api/queue').then(unwrap),
  removeFromQueue: (id) => api.post(`/api/queue/${id}/remove`).then(unwrap),
  setPriority: (id, priority) => api.post(`/api/queue/${id}/priority`, { priority }).then(unwrap),

  // Schedules
  schedules: () => api.get('/api/schedules').then(unwrap),
  schedule: (id, scheduledAt, timezone) =>
    api.post(`/api/content/${id}/schedule`, { scheduled_at: scheduledAt, timezone }).then(unwrap),
  cancelSchedule: (id) => api.delete(`/api/content/${id}/schedule`).then(unwrap),

  // Analytics
  analytics: () => api.get('/api/analytics').then(unwrap),
  analyticsHour: () => api.get('/api/analytics/hour').then(unwrap),
  analyticsPillar: () => api.get('/api/analytics/pillar').then(unwrap),

  // Topics
  topics: () => api.get('/api/topics').then(unwrap),
  createTopic: (payload) => api.post('/api/topics', payload).then(unwrap),

  // Settings
  settings: () => api.get('/api/settings').then(unwrap),
  settingsStatus: () => api.get('/api/settings/status').then(unwrap),
  updateSettings: (payload) => api.put('/api/settings', payload).then(unwrap),

  // Threads OAuth — start the login flow (redirect to Threads authorize page).
  threadsAuthURL: () => `${baseURL}/api/auth/threads`,

  // Health
  health: () => api.get('/api/health').then(unwrap),
}

export default API