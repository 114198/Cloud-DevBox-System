import api from './api'

// Types
export type GitProvider = 'github' | 'gitlab' | 'gitee' | 'bitbucket'

export type SyncStatus = 'pending' | 'syncing' | 'synced' | 'failed'

export interface GitConnection {
  id: string
  provider: GitProvider
  providerUsername: string
  scopes: string[]
  createdAt: string
  updatedAt: string
}

export interface GitRepoInfo {
  id: string
  name: string
  fullName: string
  description?: string
  cloneUrl: string
  sshUrl?: string
  defaultBranch: string
  isPrivate: boolean
  size?: number
  language?: string
}

export interface GitRepository {
  id: string
  userId: string
  connectionId: string
  environmentId?: string
  provider: GitProvider
  repoId: string
  repoName: string
  repoFullName: string
  cloneUrl: string
  sshUrl?: string
  defaultBranch: string
  isPrivate: boolean
  webhookId?: string
  lastSyncAt?: string
  syncStatus: SyncStatus
  createdAt: string
  updatedAt: string
}

export interface GitBranch {
  name: string
  commitSha: string
  isDefault: boolean
  protected: boolean
}

export interface GitCommit {
  sha: string
  message: string
  author: string
  email?: string
  timestamp: string
}

export interface GitSyncResult {
  success: boolean
  commitSha?: string
  commitMessage?: string
  filesChanged?: number
  durationMs: number
  error?: string
  syncedAt: string
}

export interface GitSyncHistory {
  id: string
  repositoryId: string
  environmentId: string
  commitSha: string
  commitMessage?: string
  commitAuthor?: string
  branch: string
  syncType: 'clone' | 'pull' | 'webhook' | 'manual'
  status: SyncStatus
  startedAt: string
  completedAt?: string
  durationMs?: number
  errorMessage?: string
}

// Request types
export interface ListRepositoriesRequest {
  provider: GitProvider
  page?: number
  pageSize?: number
  search?: string
}

export interface LinkRepositoryRequest {
  provider: GitProvider
  repoId: string
  environmentId: string
  branch?: string
  enableWebhook?: boolean
}

export interface CloneRepositoryRequest {
  repositoryId: string
  environmentId: string
  branch?: string
  shallow?: boolean
  depth?: number
}

export interface SyncRepositoryRequest {
  repositoryId: string
  environmentId: string
  branch?: string
}

// Git service
export const gitService = {
  // OAuth and connections
  getAuthUrl: async (provider: GitProvider): Promise<{ url: string; state: string }> => {
    const response = await api.get(`/git/auth/${provider}`)
    return response.data
  },

  getConnections: async (): Promise<GitConnection[]> => {
    const response = await api.get('/git/connections')
    return response.data.connections
  },

  deleteConnection: async (provider: GitProvider): Promise<void> => {
    await api.delete(`/git/connections/${provider}`)
  },

  // Repositories
  listRemoteRepositories: async (
    request: ListRepositoriesRequest
  ): Promise<{ repositories: GitRepoInfo[]; total: number }> => {
    const response = await api.get('/git/repositories', { params: request })
    return response.data
  },

  linkRepository: async (request: LinkRepositoryRequest): Promise<GitRepository> => {
    const response = await api.post('/git/repositories/link', request)
    return response.data
  },

  unlinkRepository: async (repositoryId: string): Promise<void> => {
    await api.delete(`/git/repositories/${repositoryId}`)
  },

  getLinkedRepositories: async (environmentId: string): Promise<GitRepository[]> => {
    const response = await api.get('/git/repositories/linked', {
      params: { environmentId },
    })
    return response.data.repositories
  },

  // Clone and sync
  cloneRepository: async (request: CloneRepositoryRequest): Promise<GitSyncResult> => {
    const response = await api.post('/git/clone', request)
    return response.data
  },

  syncRepository: async (request: SyncRepositoryRequest): Promise<GitSyncResult> => {
    const response = await api.post('/git/sync', request)
    return response.data
  },

  getSyncHistory: async (
    repositoryId: string,
    limit?: number
  ): Promise<GitSyncHistory[]> => {
    const response = await api.get('/git/sync/history', {
      params: { repositoryId, limit: limit || 20 },
    })
    return response.data.history
  },

  // Branches
  getBranches: async (
    provider: GitProvider,
    repoFullName: string
  ): Promise<GitBranch[]> => {
    const response = await api.get('/git/branches', {
      params: { provider, repo: repoFullName },
    })
    return response.data.branches
  },

  // Commits
  getCommits: async (
    provider: GitProvider,
    repoFullName: string,
    branch?: string,
    limit?: number
  ): Promise<GitCommit[]> => {
    const response = await api.get('/git/commits', {
      params: { provider, repo: repoFullName, branch, limit: limit || 20 },
    })
    return response.data.commits
  },

  // File diff
  getFileDiff: async (
    provider: GitProvider,
    repoFullName: string,
    commitSha: string
  ): Promise<{ files: Array<{ filename: string; status: string; additions: number; deletions: number; patch?: string }> }> => {
    const response = await api.get('/git/diff', {
      params: { provider, repo: repoFullName, commit: commitSha },
    })
    return response.data
  },
}
