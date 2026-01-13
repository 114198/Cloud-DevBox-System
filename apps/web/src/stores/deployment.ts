import { create } from 'zustand'
import { 
  Deployment, 
  DeploymentPhase, 
  DeploymentVersion, 
  DeploymentLog,
  DeploymentHistory 
} from '@/services/deployment'

interface DeploymentState {
  // Deployments list
  deployments: Deployment[]
  selectedDeployment: Deployment | null
  isLoading: boolean
  
  // Deployment history
  history: DeploymentHistory | null
  
  // Deployment logs
  logs: DeploymentLog[]
  isStreamingLogs: boolean
  
  // Actions
  setDeployments: (deployments: Deployment[]) => void
  addDeployment: (deployment: Deployment) => void
  updateDeployment: (id: string, updates: Partial<Deployment>) => void
  removeDeployment: (id: string) => void
  selectDeployment: (deployment: Deployment | null) => void
  setLoading: (loading: boolean) => void
  
  // History actions
  setHistory: (history: DeploymentHistory | null) => void
  
  // Log actions
  setLogs: (logs: DeploymentLog[]) => void
  addLog: (log: DeploymentLog) => void
  clearLogs: () => void
  setStreamingLogs: (streaming: boolean) => void
}

export const useDeploymentStore = create<DeploymentState>((set) => ({
  deployments: [],
  selectedDeployment: null,
  isLoading: false,
  history: null,
  logs: [],
  isStreamingLogs: false,

  setDeployments: (deployments) => set({ deployments }),
  
  addDeployment: (deployment) =>
    set((state) => ({
      deployments: [deployment, ...state.deployments],
    })),
  
  updateDeployment: (id, updates) =>
    set((state) => ({
      deployments: state.deployments.map((d) =>
        d.id === id ? { ...d, ...updates } : d
      ),
      selectedDeployment:
        state.selectedDeployment?.id === id
          ? { ...state.selectedDeployment, ...updates }
          : state.selectedDeployment,
    })),
  
  removeDeployment: (id) =>
    set((state) => ({
      deployments: state.deployments.filter((d) => d.id !== id),
      selectedDeployment:
        state.selectedDeployment?.id === id ? null : state.selectedDeployment,
    })),
  
  selectDeployment: (deployment) => set({ selectedDeployment: deployment }),
  
  setLoading: (isLoading) => set({ isLoading }),
  
  setHistory: (history) => set({ history }),
  
  setLogs: (logs) => set({ logs }),
  
  addLog: (log) =>
    set((state) => ({
      logs: [...state.logs, log],
    })),
  
  clearLogs: () => set({ logs: [] }),
  
  setStreamingLogs: (isStreamingLogs) => set({ isStreamingLogs }),
}))

// Helper function to get phase display info
export const getPhaseInfo = (phase: DeploymentPhase): { color: string; text: string; icon: string } => {
  const phaseMap: Record<DeploymentPhase, { color: string; text: string; icon: string }> = {
    Pending: { color: 'default', text: '等待中', icon: 'clock-circle' },
    Building: { color: 'processing', text: '构建中', icon: 'loading' },
    Pushing: { color: 'processing', text: '推送中', icon: 'cloud-upload' },
    Deploying: { color: 'processing', text: '部署中', icon: 'deployment-unit' },
    Running: { color: 'success', text: '运行中', icon: 'check-circle' },
    Failed: { color: 'error', text: '失败', icon: 'close-circle' },
    RolledBack: { color: 'warning', text: '已回滚', icon: 'rollback' },
  }
  return phaseMap[phase] || { color: 'default', text: phase, icon: 'question-circle' }
}

// Helper function to format duration
export const formatDuration = (seconds?: number): string => {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds.toFixed(1)}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}m ${remainingSeconds.toFixed(0)}s`
}
