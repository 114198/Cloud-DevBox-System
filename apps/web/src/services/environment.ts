import api from './api'
import { Environment } from '@/stores/environment'

export interface CreateEnvironmentRequest {
  name: string
  description?: string
  templateId: string
  resources?: {
    cpu: string
    memory: string
    storage: string
  }
  environment?: Array<{ name: string; value: string }>
  ports?: Array<{ containerPort: number; protocol?: string; name?: string }>
  sshPublicKey?: string
}

export interface EnvironmentStats {
  totalEnvironments: number
  runningEnvironments: number
  stoppedEnvironments: number
  suspendedEnvironments: number
  totalProjects: number
  totalCpuUsage: number
  totalMemoryUsage: number
  totalStorageUsage: number
}

export interface RecentActivity {
  id: string
  type: 'environment_created' | 'environment_started' | 'environment_stopped' | 'environment_deleted' | 'deployment' | 'collaboration'
  environmentId?: string
  environmentName?: string
  description: string
  timestamp: string
  userId: string
  userName?: string
}

export interface ResourceUsage {
  cpu: number
  memory: number
  storage: number
  network: number
  timestamp: string
}

export const environmentService = {
  list: async (): Promise<Environment[]> => {
    const response = await api.get('/environments')
    return response.data.data
  },

  get: async (id: string): Promise<Environment> => {
    const response = await api.get(`/environments/${id}`)
    return response.data.data
  },

  create: async (data: CreateEnvironmentRequest): Promise<Environment> => {
    const response = await api.post('/environments', data)
    return response.data.data
  },

  update: async (id: string, data: Partial<CreateEnvironmentRequest>): Promise<Environment> => {
    const response = await api.put(`/environments/${id}`, data)
    return response.data.data
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/environments/${id}`)
  },

  start: async (id: string): Promise<void> => {
    await api.post(`/environments/${id}/start`)
  },

  stop: async (id: string): Promise<void> => {
    await api.post(`/environments/${id}/stop`)
  },

  restart: async (id: string): Promise<void> => {
    await api.post(`/environments/${id}/restart`)
  },

  // Dashboard statistics
  getStats: async (): Promise<EnvironmentStats> => {
    const response = await api.get('/environments/stats')
    return response.data.data
  },

  // Recent activities
  getRecentActivities: async (limit?: number): Promise<RecentActivity[]> => {
    const response = await api.get('/environments/activities', {
      params: { limit: limit || 10 }
    })
    return response.data.data
  },

  // Resource usage for a specific environment
  getResourceUsage: async (id: string): Promise<ResourceUsage> => {
    const response = await api.get(`/environments/${id}/resources`)
    return response.data.data
  },

  // Resource usage history for charts
  getResourceHistory: async (id: string, period?: string): Promise<ResourceUsage[]> => {
    const response = await api.get(`/environments/${id}/resources/history`, {
      params: { period: period || '1h' }
    })
    return response.data.data
  },

  // Logs for an environment
  getLogs: async (id: string, lines?: number): Promise<string[]> => {
    const response = await api.get(`/environments/${id}/logs`, {
      params: { lines: lines || 100 }
    })
    return response.data.data
  },

  // Batch operations
  batchStart: async (ids: string[]): Promise<void> => {
    await api.post('/environments/batch/start', { ids })
  },

  batchStop: async (ids: string[]): Promise<void> => {
    await api.post('/environments/batch/stop', { ids })
  },

  batchDelete: async (ids: string[]): Promise<void> => {
    await api.post('/environments/batch/delete', { ids })
  },
}
