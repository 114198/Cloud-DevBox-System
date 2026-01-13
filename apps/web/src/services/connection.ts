import api from './api'

// SSH Key types
export type SSHKeyType = 'ed25519' | 'rsa'

export interface SSHKeyPair {
  id: string
  userId: string
  name: string
  type: SSHKeyType
  publicKey: string
  fingerprint: string
  comment?: string
  createdAt: string
  expiresAt?: string
  lastUsedAt?: string
  rotatedAt?: string
  isDefault: boolean
}

export interface SSHConfig {
  host: string
  port: number
  user: string
  identityFile?: string
  proxyCommand?: string
  strictHostKeyChecking?: string
  userKnownHostsFile?: string
  serverAliveInterval?: number
  serverAliveCountMax?: number
}

export type IDEType = 'vscode' | 'jetbrains' | 'generic'

export interface SSHConfigResponse {
  config: SSHConfig
  configString: string
  ideType: IDEType
  instructions: string
  generationTime?: number
}

export interface SSHConnection {
  id: string
  environmentId: string
  userId: string
  keyId: string
  clientIp: string
  clientPort: number
  serverPort: number
  connectedAt: string
  lastActivityAt: string
  bytesSent: number
  bytesReceived: number
}

export interface SSHConnectionStats {
  environmentId: string
  activeConnections: number
  maxConnections: number
  totalConnections: number
  totalBytesSent: number
  totalBytesReceived: number
}

export interface ProxyConfig {
  gatewayHost: string
  gatewayPort: number
  gatewayUser: string
  targetHost: string
  targetPort: number
  connectionReuse: boolean
  keepAlive: number
}

export interface CreateSSHKeyRequest {
  name: string
  type: SSHKeyType
  comment?: string
  expiresIn?: number
  setDefault?: boolean
}

export const connectionService = {
  // SSH Key Management
  generateKeyPair: async (data: CreateSSHKeyRequest): Promise<{ key: SSHKeyPair; privateKey: string }> => {
    const response = await api.post('/ssh/keys', data)
    return response.data
  },

  listKeys: async (page = 1, pageSize = 20): Promise<{ keys: SSHKeyPair[]; total: number }> => {
    const response = await api.get('/ssh/keys', { params: { page, pageSize } })
    return response.data
  },

  getKey: async (keyId: string): Promise<SSHKeyPair> => {
    const response = await api.get(`/ssh/keys/${keyId}`)
    return response.data
  },

  deleteKey: async (keyId: string): Promise<void> => {
    await api.delete(`/ssh/keys/${keyId}`)
  },

  rotateKey: async (keyId: string): Promise<{ key: SSHKeyPair; privateKey: string }> => {
    const response = await api.post(`/ssh/keys/${keyId}/rotate`)
    return response.data
  },

  setDefaultKey: async (keyId: string): Promise<void> => {
    await api.post(`/ssh/keys/${keyId}/default`)
  },

  // SSH Config Generation
  getSSHConfig: async (envId: string, ideType: IDEType = 'generic', keyPath?: string): Promise<SSHConfigResponse> => {
    const params: Record<string, string> = { ideType }
    if (keyPath) params.keyPath = keyPath
    const response = await api.get(`/environments/${envId}/ssh/config`, { params })
    return response.data.config ? response.data : response.data
  },

  getSSHCommand: async (envId: string, keyPath?: string): Promise<string> => {
    const params: Record<string, string> = {}
    if (keyPath) params.keyPath = keyPath
    const response = await api.get(`/environments/${envId}/ssh/command`, { params })
    return response.data.command
  },

  getProxyConfig: async (envId: string): Promise<ProxyConfig> => {
    const response = await api.get(`/environments/${envId}/ssh/proxy`)
    return response.data
  },

  // Connection Management
  getConnectionStats: async (envId: string): Promise<SSHConnectionStats> => {
    const response = await api.get(`/environments/${envId}/ssh/stats`)
    return response.data
  },

  getConnections: async (envId: string): Promise<{ connections: SSHConnection[]; count: number }> => {
    const response = await api.get(`/environments/${envId}/ssh/connections`)
    return response.data
  },

  canConnect: async (envId: string): Promise<{ canConnect: boolean; currentConnections: number; maxConnections: number }> => {
    const response = await api.get(`/environments/${envId}/ssh/can-connect`)
    return response.data
  },
}
