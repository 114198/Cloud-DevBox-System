import api from './api'

// Deployment types
export type DeploymentPhase = 
  | 'Pending' 
  | 'Building' 
  | 'Pushing' 
  | 'Deploying' 
  | 'Running' 
  | 'Failed' 
  | 'RolledBack'

export interface BuildConfig {
  dockerfilePath?: string
  contextPath?: string
  buildArgs?: Record<string, string>
  target?: string
  noCache?: boolean
  platform?: string
  sourceType?: string
  gitUrl?: string
  gitBranch?: string
  gitCommit?: string
}

export interface ResourceConfig {
  cpu: string
  memory: string
  storage?: string
}

export interface EnvVar {
  name: string
  value: string
}

export interface PortMapping {
  name: string
  containerPort: number
  protocol?: string
}

export interface HealthCheckConfig {
  path?: string
  port?: number
  initialDelaySeconds?: number
  periodSeconds?: number
  timeoutSeconds?: number
  failureThreshold?: number
  successThreshold?: number
}

export interface DeploymentStrategy {
  type?: string
  maxUnavailable?: string
  maxSurge?: string
}

export interface IngressConfig {
  enabled?: boolean
  host?: string
  path?: string
  tls?: boolean
  tlsSecretName?: string
  annotations?: Record<string, string>
}

export interface AutoScalingConfig {
  enabled?: boolean
  minReplicas?: number
  maxReplicas?: number
  targetCPUUtilization?: number
  targetMemoryUtilization?: number
}

export interface DeployConfig {
  replicas?: number
  resources?: ResourceConfig
  environment?: EnvVar[]
  ports?: PortMapping[]
  healthCheck?: HealthCheckConfig
  strategy?: DeploymentStrategy
  serviceType?: string
  ingress?: IngressConfig
  autoScaling?: AutoScalingConfig
}

export interface ImageInfo {
  name: string
  tag: string
  digest?: string
  size?: number
  buildDuration?: number
  pushDuration?: number
  registry?: string
  createdAt: string
}

export interface K8sResourceInfo {
  deploymentName?: string
  serviceName?: string
  ingressName?: string
  hpaName?: string
  namespace?: string
  externalUrl?: string
  internalUrl?: string
}

export interface DeploymentMetrics {
  buildTime?: number
  pushTime?: number
  deployTime?: number
  totalTime?: number
  retries?: number
}

export interface Deployment {
  id: string
  environmentId: string
  userId: string
  name: string
  version: string
  phase: DeploymentPhase
  message?: string
  buildConfig?: BuildConfig
  deployConfig?: DeployConfig
  imageInfo?: ImageInfo
  k8sResources?: K8sResourceInfo
  metrics?: DeploymentMetrics
  createdAt: string
  updatedAt: string
  startedAt?: string
  completedAt?: string
}

export interface DeploymentVersion {
  id: string
  deploymentId: string
  version: string
  imageInfo?: ImageInfo
  deployConfig?: DeployConfig
  phase: DeploymentPhase
  message?: string
  createdAt: string
  deployedAt?: string
  rolledBackAt?: string
  isActive: boolean
}

export interface DeploymentLog {
  id: string
  deploymentId: string
  phase: string
  level: 'info' | 'warn' | 'error'
  message: string
  timestamp: string
  details?: string
}

export interface DeploymentHistory {
  deploymentId: string
  versions: DeploymentVersion[]
  total: number
}

export interface CreateDeploymentRequest {
  environmentId: string
  name: string
  buildConfig?: BuildConfig
  deployConfig?: DeployConfig
}

export interface RollbackRequest {
  targetVersion: string
  reason?: string
}

export interface ListDeploymentsRequest {
  environmentId?: string
  userId?: string
  phase?: DeploymentPhase
  page?: number
  pageSize?: number
}

export interface ListDeploymentsResponse {
  deployments: Deployment[]
  total: number
  page: number
  pageSize: number
}

export interface DockerfileTemplate {
  name: string
  language: string
  framework?: string
  content: string
  buildArgs?: Record<string, string>
  description?: string
}

export interface GenerateDockerfileRequest {
  language: string
  framework?: string
  version?: string
  buildArgs?: Record<string, string>
  multiStage?: boolean
  outputPath?: string
}

export interface GenerateDockerfileResponse {
  dockerfile: string
  path?: string
}

export const deploymentService = {
  // Create a new deployment
  create: async (data: CreateDeploymentRequest): Promise<Deployment> => {
    const response = await api.post('/deployments', data)
    return response.data
  },

  // Get a deployment by ID
  get: async (id: string): Promise<Deployment> => {
    const response = await api.get(`/deployments/${id}`)
    return response.data
  },

  // List deployments
  list: async (params?: ListDeploymentsRequest): Promise<ListDeploymentsResponse> => {
    const response = await api.get('/deployments', { params })
    return response.data
  },

  // Delete a deployment
  delete: async (id: string): Promise<void> => {
    await api.delete(`/deployments/${id}`)
  },

  // Rollback a deployment
  rollback: async (id: string, data: RollbackRequest): Promise<Deployment> => {
    const response = await api.post(`/deployments/${id}/rollback`, data)
    return response.data
  },

  // Get deployment logs
  getLogs: async (id: string): Promise<DeploymentLog[]> => {
    const response = await api.get(`/deployments/${id}/logs`)
    return response.data.logs
  },

  // Get deployment history
  getHistory: async (id: string): Promise<DeploymentHistory> => {
    const response = await api.get(`/deployments/${id}/history`)
    return response.data
  },

  // Generate Dockerfile
  generateDockerfile: async (data: GenerateDockerfileRequest): Promise<GenerateDockerfileResponse> => {
    const response = await api.post('/deployments/dockerfile/generate', data)
    return response.data
  },

  // Get Dockerfile templates
  getDockerfileTemplates: async (): Promise<DockerfileTemplate[]> => {
    const response = await api.get('/deployments/dockerfile/templates')
    return response.data.templates
  },

  // Stream logs (returns EventSource URL)
  getLogsStreamUrl: (id: string): string => {
    return `/api/v1/deployments/${id}/logs/stream`
  },
}
