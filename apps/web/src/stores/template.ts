import { create } from 'zustand'
import { Template, TemplateListParams } from '@/services/template'

interface TemplateState {
  // Template list state
  templates: Template[]
  total: number
  page: number
  pageSize: number
  totalPages: number
  isLoading: boolean
  error: string | null

  // Filter state
  filters: TemplateListParams
  
  // Selected template for configuration
  selectedTemplate: Template | null

  // Actions
  setTemplates: (templates: Template[], total: number, page: number, pageSize: number, totalPages: number) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
  setFilters: (filters: Partial<TemplateListParams>) => void
  resetFilters: () => void
  selectTemplate: (template: Template | null) => void
  setPage: (page: number) => void
}

const defaultFilters: TemplateListParams = {
  category: undefined,
  tags: undefined,
  search: undefined,
  is_public: true,
  page: 1,
  page_size: 20,
}

export const useTemplateStore = create<TemplateState>((set) => ({
  templates: [],
  total: 0,
  page: 1,
  pageSize: 20,
  totalPages: 0,
  isLoading: false,
  error: null,
  filters: { ...defaultFilters },
  selectedTemplate: null,

  setTemplates: (templates, total, page, pageSize, totalPages) =>
    set({ templates, total, page, pageSize, totalPages, error: null }),

  setLoading: (isLoading) => set({ isLoading }),

  setError: (error) => set({ error, isLoading: false }),

  setFilters: (newFilters) =>
    set((state) => ({
      filters: { ...state.filters, ...newFilters, page: 1 },
    })),

  resetFilters: () => set({ filters: { ...defaultFilters } }),

  selectTemplate: (selectedTemplate) => set({ selectedTemplate }),

  setPage: (page) =>
    set((state) => ({
      filters: { ...state.filters, page },
    })),
}))

// Selector hooks
export const useTemplates = () => useTemplateStore((state) => state.templates)
export const useSelectedTemplate = () => useTemplateStore((state) => state.selectedTemplate)
export const useTemplateFilters = () => useTemplateStore((state) => state.filters)
export const useTemplateLoading = () => useTemplateStore((state) => state.isLoading)
