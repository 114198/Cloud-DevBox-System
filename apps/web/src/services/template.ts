import api from './api'

// Template types matching backend models
export interface TemplateRuntime {
  language: string
  version: string
  framework?: string
  dependencies?: Array<{ name: string; version: string }>
  buildTools?: string[]
}

export interface TemplateDefaultConfig {
  cpu: string
  memory: string
  storage: string
  image?: string
  ports?: Array<{ containerPort: number; protocol?: string }>
  environment?: Record<string, string>
}

export interface Template {
  id: string
  name: string
  display_name: string
  description?: string
  category: string
  tags: string[]
  icon?: string
  runtime: TemplateRuntime
  default_config: TemplateDefaultConfig
  dockerfile: string
  init_script?: string
  extensions?: Array<{ name: string; version: string; config?: Record<string, unknown> }>
  is_public: boolean
  created_by: string
  version: string
  parent_id?: string
  created_at: string
  updated_at: string
}

export interface TemplateListResponse {
  data: Template[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface TemplateListParams {
  category?: string
  tags?: string[]
  search?: string
  is_public?: boolean
  page?: number
  page_size?: number
}

export interface CategoryInfo {
  name: string
  count: number
}

// Template API service
export const templateService = {
  // List templates with filtering and pagination
  async list(params: TemplateListParams = {}): Promise<TemplateListResponse> {
    const queryParams = new URLSearchParams()
    
    if (params.category) queryParams.append('category', params.category)
    if (params.search) queryParams.append('search', params.search)
    if (params.is_public !== undefined) queryParams.append('is_public', String(params.is_public))
    if (params.page) queryParams.append('page', String(params.page))
    if (params.page_size) queryParams.append('page_size', String(params.page_size))
    if (params.tags?.length) {
      params.tags.forEach(tag => queryParams.append('tags', tag))
    }
    
    const response = await api.get<TemplateListResponse>(`/templates?${queryParams.toString()}`)
    return response.data
  },

  // Get a single template by ID
  async getById(id: string): Promise<Template> {
    const response = await api.get<{ data: Template }>(`/templates/${id}`)
    return response.data.data
  },

  // Get all categories with counts
  async getCategories(): Promise<CategoryInfo[]> {
    const response = await api.get<{ data: CategoryInfo[] }>('/templates/categories')
    return response.data.data
  },

  // Get valid categories (enum values)
  async getValidCategories(): Promise<string[]> {
    const response = await api.get<{ data: string[] }>('/templates/valid-categories')
    return response.data.data
  },

  // Get all tags
  async getTags(): Promise<string[]> {
    const response = await api.get<{ data: string[] }>('/templates/tags')
    return response.data.data
  },
}

export default templateService
