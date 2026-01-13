import api from './api'

// Backup types
export type BackupType = 'code' | 'config' | 'database' | 'full'
export type BackupStatus = 'pending' | 'in_progress' | 'completed' | 'failed'
export type BackupMode = 'full' | 'incremental'

export interface BackupMetadata {
  fileCount?: number
  totalSize?: number
  changedFiles?: number
  deletedFiles?: number
  sourcePath?: string
  compression?: string
  encryption?: string
  labels?: Record<string, string>
}

export interface Backup {
  id: string
  environmentId: string
  userId: string
  type: BackupType
  mode: BackupMode
  status: BackupStatus
  size: number
  storagePath: string
  storageRegion: string
  checksum: string
  parentId?: string
  message?: string
  metadata: BackupMetadata
  createdAt: string
  completedAt?: string
  expiresAt?: string
}

export interface BackupStats {
  totalBackups: number
  totalSize: number
  codeBackups: number
  configBackups: number
  databaseBackups: number
  lastBackupAt?: string
  oldestBackupAt?: string
}

export interface ConfigVersion {
  id: string
  environmentId: string
  version: number
  config: Record<string, unknown>
  message?: string
  createdAt: string
  createdBy: string
}

export interface ConfigDiff {
  fromVersion: number
  toVersion: number
  changes: Array<{
    path: string
    type: 'added' | 'removed' | 'modified'
    oldValue?: unknown
    newValue?: unknown
  }>
}

export interface RestoreRecord {
  id: string
  backupId: string
  environmentId: string
  userId: string
  status: 'pending' | 'in_progress' | 'completed' | 'failed'
  message?: string
  restoredFiles: number
  restoredSize: number
  startedAt: string
  completedAt?: string
}

export interface ListBackupsResponse {
  backups: Backup[]
  total: number
  page: number
  pageSize: number
}

export interface ListConfigVersionsResponse {
  versions: ConfigVersion[]
  total: number
  page: number
  pageSize: number
}

// Environment config export/import types
export interface EnvironmentExportConfig {
  name: string
  description?: string
  templateId: string
  resources: {
    cpu: string
    memory: string
    storage: string
  }
  environment: Record<string, string>
  ports: Array<{
    containerPort: number
    protocol: string
    name?: string
  }>
  volumes: Array<{
    name: string
    mountPath: string
    size: string
  }>
  runtime: {
    language: string
    version: string
    framework?: string
  }
  exportedAt: string
  exportedBy: string
  version: string
}

export const backupService = {
  // Code backups
  createCodeBackup: async (environmentId: string, mode: BackupMode = 'incremental'): Promise<Backup> => {
    const response = await api.post('/backups/code', {
      environmentId,
      type: 'code',
      mode,
    })
    return response.data
  },

  listCodeBackups: async (params?: {
    environmentId?: string
    status?: BackupStatus
    page?: number
    pageSize?: number
  }): Promise<ListBackupsResponse> => {
    const response = await api.get('/backups/code', { params })
    return response.data
  },

  getCodeBackup: async (id: string): Promise<Backup> => {
    const response = await api.get(`/backups/code/${id}`)
    return response.data
  },

  deleteCodeBackup: async (id: string): Promise<void> => {
    await api.delete(`/backups/code/${id}`)
  },

  getCodeBackupStats: async (): Promise<BackupStats> => {
    const response = await api.get('/backups/code/stats')
    return response.data
  },

  // Config backups (version control)
  saveConfig: async (
    environmentId: string,
    config: Record<string, unknown>,
    message?: string
  ): Promise<ConfigVersion> => {
    const response = await api.post(`/environments/${environmentId}/config/versions`, {
      config,
      message,
    })
    return response.data
  },

  listConfigVersions: async (
    environmentId: string,
    page?: number,
    pageSize?: number
  ): Promise<ListConfigVersionsResponse> => {
    const response = await api.get(`/environments/${environmentId}/config/versions`, {
      params: { page, pageSize },
    })
    return response.data
  },

  getConfigVersion: async (environmentId: string, version: number): Promise<ConfigVersion> => {
    const response = await api.get(`/environments/${environmentId}/config/versions/${version}`)
    return response.data
  },

  getLatestConfigVersion: async (environmentId: string): Promise<ConfigVersion> => {
    const response = await api.get(`/environments/${environmentId}/config/versions/latest`)
    return response.data
  },

  compareConfigVersions: async (
    environmentId: string,
    fromVersion: number,
    toVersion: number
  ): Promise<ConfigDiff> => {
    const response = await api.get(`/environments/${environmentId}/config/compare`, {
      params: { from: fromVersion, to: toVersion },
    })
    return response.data
  },

  restoreConfigVersion: async (environmentId: string, version: number): Promise<ConfigVersion> => {
    const response = await api.post(`/environments/${environmentId}/config/versions/${version}/restore`)
    return response.data
  },

  getConfigBackupStats: async (): Promise<BackupStats> => {
    const response = await api.get('/backups/config/stats')
    return response.data
  },

  // Database backups
  createDatabaseBackup: async (mode: BackupMode = 'incremental'): Promise<Backup> => {
    const response = await api.post('/backups/database', { mode })
    return response.data
  },

  listDatabaseBackups: async (page?: number, pageSize?: number): Promise<ListBackupsResponse> => {
    const response = await api.get('/backups/database', {
      params: { page, pageSize },
    })
    return response.data
  },

  getDatabaseBackup: async (id: string): Promise<Backup> => {
    const response = await api.get(`/backups/database/${id}`)
    return response.data
  },

  deleteDatabaseBackup: async (id: string): Promise<void> => {
    await api.delete(`/backups/database/${id}`)
  },

  restoreDatabaseBackup: async (id: string): Promise<void> => {
    await api.post(`/backups/database/${id}/restore`)
  },

  verifyDatabaseBackup: async (id: string): Promise<{ valid: boolean }> => {
    const response = await api.get(`/backups/database/${id}/verify`)
    return response.data
  },

  getDatabaseBackupStats: async (): Promise<BackupStats> => {
    const response = await api.get('/backups/database/stats')
    return response.data
  },

  // Restore operations
  restoreFromBackup: async (backupId: string, environmentId: string, overwrite?: boolean): Promise<RestoreRecord> => {
    const response = await api.post('/backups/restore', {
      backupId,
      environmentId,
      overwrite,
    })
    return response.data
  },

  // Environment config export/import
  exportEnvironmentConfig: async (environmentId: string): Promise<EnvironmentExportConfig> => {
    const response = await api.get(`/environments/${environmentId}/export`)
    return response.data
  },

  importEnvironmentConfig: async (config: EnvironmentExportConfig): Promise<{ environmentId: string }> => {
    const response = await api.post('/environments/import', config)
    return response.data
  },

  downloadEnvironmentConfigYaml: async (environmentId: string): Promise<string> => {
    const response = await api.get(`/environments/${environmentId}/export/yaml`, {
      responseType: 'text',
    })
    return response.data
  },
}
