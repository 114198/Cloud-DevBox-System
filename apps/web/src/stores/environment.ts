import { create } from 'zustand'

export type EnvironmentStatus = 'creating' | 'running' | 'stopped' | 'failed' | 'suspended'

export interface Environment {
  id: string
  name: string
  description?: string
  templateId: string
  templateName: string
  status: EnvironmentStatus
  resources: {
    cpu: string
    memory: string
    storage: string
  }
  createdAt: string
  lastAccessedAt?: string
}

interface EnvironmentState {
  environments: Environment[]
  selectedEnvironment: Environment | null
  isLoading: boolean
  setEnvironments: (environments: Environment[]) => void
  selectEnvironment: (environment: Environment | null) => void
  addEnvironment: (environment: Environment) => void
  updateEnvironment: (id: string, updates: Partial<Environment>) => void
  removeEnvironment: (id: string) => void
  setLoading: (loading: boolean) => void
}

export const useEnvironmentStore = create<EnvironmentState>((set) => ({
  environments: [],
  selectedEnvironment: null,
  isLoading: false,
  setEnvironments: (environments) => set({ environments }),
  selectEnvironment: (environment) => set({ selectedEnvironment: environment }),
  addEnvironment: (environment) =>
    set((state) => ({
      environments: [...state.environments, environment],
    })),
  updateEnvironment: (id, updates) =>
    set((state) => ({
      environments: state.environments.map((env) =>
        env.id === id ? { ...env, ...updates } : env
      ),
    })),
  removeEnvironment: (id) =>
    set((state) => ({
      environments: state.environments.filter((env) => env.id !== id),
    })),
  setLoading: (isLoading) => set({ isLoading }),
}))
