import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import API from '../services/api'

export const useDashboard = () =>
  useQuery({ queryKey: ['dashboard'], queryFn: API.dashboard })

export const useContentList = (params) =>
  useQuery({ queryKey: ['content', params], queryFn: () => API.listContent(params) })

export const useContent = (id) =>
  useQuery({ queryKey: ['content', id], queryFn: () => API.getContent(id), enabled: !!id })

export const usePreview = (id) =>
  useQuery({ queryKey: ['preview', id], queryFn: () => API.getPreview(id), enabled: !!id })

export const useQueue = () => useQuery({ queryKey: ['queue'], queryFn: API.queue })

export const useSchedules = () => useQuery({ queryKey: ['schedules'], queryFn: API.schedules })

export const useAnalytics = () => useQuery({ queryKey: ['analytics'], queryFn: API.analytics })
export const useAnalyticsHour = () =>
  useQuery({ queryKey: ['analytics-hour'], queryFn: API.analyticsHour })
export const useAnalyticsPillar = () =>
  useQuery({ queryKey: ['analytics-pillar'], queryFn: API.analyticsPillar })

export const useTopics = () => useQuery({ queryKey: ['topics'], queryFn: API.topics })

export const useSettings = () => useQuery({ queryKey: ['settings'], queryFn: API.settings })

const invalidate = (qc, keys) => keys.forEach((k) => qc.invalidateQueries({ queryKey: k }))

export const useGenerate = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => API.generate(payload),
    onSuccess: () => invalidate(qc, [['dashboard'], ['content']]),
  })
}

export const useRegenerate = (id) => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => API.regenerate(id, payload),
    onSuccess: () => invalidate(qc, [['content'], ['preview', id]]),
  })
}

export const useUpdateContent = (id) => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => API.updateContent(id, payload),
    onSuccess: () => invalidate(qc, [['content'], ['preview', id]]),
  })
}

export const useApprove = (id) => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => API.approve(id),
    onSuccess: () => invalidate(qc, [['dashboard'], ['content'], ['queue'], ['preview', id]]),
  })
}

export const useReject = (id) => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (reason) => API.reject(id, reason),
    onSuccess: () => invalidate(qc, [['content'], ['preview', id], ['dashboard']]),
  })
}

export const useRemoveFromQueue = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id) => API.removeFromQueue(id),
    onSuccess: () => invalidate(qc, [['queue'], ['dashboard'], ['content']]),
  })
}

export const useSetPriority = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, priority }) => API.setPriority(id, priority),
    onSuccess: () => invalidate(qc, [['queue']]),
  })
}

export const useSchedule = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, scheduledAt, timezone }) => API.schedule(id, scheduledAt, timezone),
    onSuccess: () => invalidate(qc, [['schedules'], ['dashboard'], ['content']]),
  })
}

export const useCancelSchedule = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id) => API.cancelSchedule(id),
    onSuccess: () => invalidate(qc, [['schedules'], ['dashboard'], ['content']]),
  })
}

export const useCreateTopic = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => API.createTopic(payload),
    onSuccess: () => invalidate(qc, [['topics']]),
  })
}

export const useUpdateSettings = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload) => API.updateSettings(payload),
    onSuccess: () => invalidate(qc, [['settings']]),
  })
}