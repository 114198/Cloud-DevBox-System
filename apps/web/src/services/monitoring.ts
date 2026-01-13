import api from './api'

// Metric data point
export interface MetricDataPoint {
  timestamp: string
  value: number
}

// Environment metrics
export interface EnvironmentMetrics {
  environmentId: string
  timestamp: string
  cpuUsage: number
  cpuCores: number
  cpuLimit: number
  memoryUsage: number
  memoryUsed: number
  memoryLimit: number
  storageUsage: number
  storageUsed: number
  storageLimit: number
  networkRxBytes: number
  networkTxBytes: number
  networkRxRate: number
  networkTxRate: number
}

// Metrics history
export interface MetricsHistory {
  environmentId: string
  period: string
  resolution: string
  startTime: string
  endTime: string
  cpu: MetricDataPoint[]
  memory: MetricDataPoint[]
  storage: MetricDataPoint[]
  networkRx: MetricDataPoint[]
  networkTx: MetricDataPoint[]
}

// Alert severity
export type AlertSeverity = 'info' | 'warning' | 'critical'

// Alert status
export type AlertStatus = 'active' | 'resolved' | 'acknowledged'

// Alert
export interface Alert {
  id: string
  environmentId: string
  userId: string
  type: 'cpu' | 'memory' | 'storage' | 'network'
  severity: AlertSeverity
  status: AlertStatus
  title: string
  message: string
  value: number
  threshold: number
  createdAt: string
  updatedAt: string
  resolvedAt?: string
  ackedAt?: string
  ackedBy?: string
  notifiedAt?: string
  notifyMethods?: string[]
}

// Alert rule
export interface AlertRule {
  id: string
  userId: string
  environmentId?: string
  type: 'cpu' | 'memory' | 'storage' | 'network'
  severity: AlertSeverity
  threshold: number
  duration: number
  enabled: boolean
  notifyMethods: string[]
  createdAt: string
  updatedAt: string
}


// Alert statistics
export interface AlertStats {
  totalAlerts: number
  activeAlerts: number
  resolvedAlerts: number
  criticalAlerts: number
  warningAlerts: number
  infoAlerts: number
}

// List alerts request
export interface ListAlertsRequest {
  environmentId?: string
  status?: AlertStatus
  severity?: AlertSeverity
  page?: number
  pageSize?: number
}

// List alerts response
export interface ListAlertsResponse {
  alerts: Alert[]
  total: number
  page: number
  pageSize: number
}

// Create alert rule request
export interface CreateAlertRuleRequest {
  environmentId?: string
  type: 'cpu' | 'memory' | 'storage' | 'network'
  severity: AlertSeverity
  threshold: number
  duration: string
  notifyMethods: string[]
}

// Update alert rule request
export interface UpdateAlertRuleRequest {
  threshold?: number
  duration?: string
  enabled?: boolean
  notifyMethods?: string[]
}

export const monitoringService = {
  // Get current metrics for an environment
  getMetrics: async (environmentId: string): Promise<EnvironmentMetrics> => {
    const response = await api.get(`/environments/${environmentId}/metrics`)
    return response.data.data
  },

  // Get metrics history for an environment
  getMetricsHistory: async (
    environmentId: string,
    period?: string,
    resolution?: string
  ): Promise<MetricsHistory> => {
    const response = await api.get(`/environments/${environmentId}/metrics/history`, {
      params: { period: period || '1h', resolution: resolution || '1m' }
    })
    return response.data.data
  },

  // List alerts
  listAlerts: async (params?: ListAlertsRequest): Promise<ListAlertsResponse> => {
    const response = await api.get('/alerts', { params })
    return response.data.data
  },

  // Get alert by ID
  getAlert: async (alertId: string): Promise<Alert> => {
    const response = await api.get(`/alerts/${alertId}`)
    return response.data.data
  },

  // Acknowledge alert
  acknowledgeAlert: async (alertId: string): Promise<Alert> => {
    const response = await api.post(`/alerts/${alertId}/acknowledge`)
    return response.data.data
  },

  // Get alert statistics
  getAlertStats: async (): Promise<AlertStats> => {
    const response = await api.get('/alerts/stats')
    return response.data.data
  },

  // List alert rules
  listAlertRules: async (): Promise<AlertRule[]> => {
    const response = await api.get('/alerts/rules')
    return response.data.data
  },

  // Create alert rule
  createAlertRule: async (data: CreateAlertRuleRequest): Promise<AlertRule> => {
    const response = await api.post('/alerts/rules', data)
    return response.data.data
  },

  // Update alert rule
  updateAlertRule: async (ruleId: string, data: UpdateAlertRuleRequest): Promise<AlertRule> => {
    const response = await api.put(`/alerts/rules/${ruleId}`, data)
    return response.data.data
  },

  // Delete alert rule
  deleteAlertRule: async (ruleId: string): Promise<void> => {
    await api.delete(`/alerts/rules/${ruleId}`)
  },
}
