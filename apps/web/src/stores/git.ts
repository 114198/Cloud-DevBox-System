import { create } from 'zustand'
import {
  GitConnection,
  GitRepository,
  GitRepoInfo,
  GitBranch,
  GitCommit,
  GitSyncHistory,
  GitProvider,
} from '@/services/git'

interface GitState {
  // Connections
  connections: GitConnection[]
  isLoadingConnections: boolean
  
  // Remote repositories (from provider)
  remoteRepositories: GitRepoInfo[]
  remoteReposTotal: number
  isLoadingRemoteRepos: boolean
  
  // Linked repositories
  linkedRepositories: GitRepository[]
  isLoadingLinkedRepos: boolean
  
  // Selected repository for detail view
  selectedRepository: GitRepository | null
  
  // Branches and commits for selected repo
  branches: GitBranch[]
  commits: GitCommit[]
  isLoadingBranches: boolean
  isLoadingCommits: boolean
  
  // Sync history
  syncHistory: GitSyncHistory[]
  isLoadingSyncHistory: boolean
  
  // Current provider filter
  currentProvider: GitProvider | null
  
  // Actions
  setConnections: (connections: GitConnection[]) => void
  addConnection: (connection: GitConnection) => void
  removeConnection: (provider: GitProvider) => void
  setLoadingConnections: (loading: boolean) => void
  
  setRemoteRepositories: (repos: GitRepoInfo[], total: number) => void
  setLoadingRemoteRepos: (loading: boolean) => void
  
  setLinkedRepositories: (repos: GitRepository[]) => void
  addLinkedRepository: (repo: GitRepository) => void
  removeLinkedRepository: (repoId: string) => void
  updateLinkedRepository: (repoId: string, updates: Partial<GitRepository>) => void
  setLoadingLinkedRepos: (loading: boolean) => void
  
  setSelectedRepository: (repo: GitRepository | null) => void
  
  setBranches: (branches: GitBranch[]) => void
  setLoadingBranches: (loading: boolean) => void
  
  setCommits: (commits: GitCommit[]) => void
  setLoadingCommits: (loading: boolean) => void
  
  setSyncHistory: (history: GitSyncHistory[]) => void
  setLoadingSyncHistory: (loading: boolean) => void
  
  setCurrentProvider: (provider: GitProvider | null) => void
  
  reset: () => void
}

const initialState = {
  connections: [],
  isLoadingConnections: false,
  remoteRepositories: [],
  remoteReposTotal: 0,
  isLoadingRemoteRepos: false,
  linkedRepositories: [],
  isLoadingLinkedRepos: false,
  selectedRepository: null,
  branches: [],
  commits: [],
  isLoadingBranches: false,
  isLoadingCommits: false,
  syncHistory: [],
  isLoadingSyncHistory: false,
  currentProvider: null,
}

export const useGitStore = create<GitState>((set) => ({
  ...initialState,

  // Connection actions
  setConnections: (connections) => set({ connections }),
  addConnection: (connection) =>
    set((state) => ({
      connections: [...state.connections.filter(c => c.provider !== connection.provider), connection],
    })),
  removeConnection: (provider) =>
    set((state) => ({
      connections: state.connections.filter((c) => c.provider !== provider),
    })),
  setLoadingConnections: (isLoadingConnections) => set({ isLoadingConnections }),

  // Remote repositories actions
  setRemoteRepositories: (remoteRepositories, remoteReposTotal) =>
    set({ remoteRepositories, remoteReposTotal }),
  setLoadingRemoteRepos: (isLoadingRemoteRepos) => set({ isLoadingRemoteRepos }),

  // Linked repositories actions
  setLinkedRepositories: (linkedRepositories) => set({ linkedRepositories }),
  addLinkedRepository: (repo) =>
    set((state) => ({
      linkedRepositories: [...state.linkedRepositories, repo],
    })),
  removeLinkedRepository: (repoId) =>
    set((state) => ({
      linkedRepositories: state.linkedRepositories.filter((r) => r.id !== repoId),
    })),
  updateLinkedRepository: (repoId, updates) =>
    set((state) => ({
      linkedRepositories: state.linkedRepositories.map((r) =>
        r.id === repoId ? { ...r, ...updates } : r
      ),
    })),
  setLoadingLinkedRepos: (isLoadingLinkedRepos) => set({ isLoadingLinkedRepos }),

  // Selected repository
  setSelectedRepository: (selectedRepository) => set({ selectedRepository }),

  // Branches
  setBranches: (branches) => set({ branches }),
  setLoadingBranches: (isLoadingBranches) => set({ isLoadingBranches }),

  // Commits
  setCommits: (commits) => set({ commits }),
  setLoadingCommits: (isLoadingCommits) => set({ isLoadingCommits }),

  // Sync history
  setSyncHistory: (syncHistory) => set({ syncHistory }),
  setLoadingSyncHistory: (isLoadingSyncHistory) => set({ isLoadingSyncHistory }),

  // Provider filter
  setCurrentProvider: (currentProvider) => set({ currentProvider }),

  // Reset
  reset: () => set(initialState),
}))
