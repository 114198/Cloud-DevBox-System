/**
 * Templates Service
 */

import type { DevBoxClient } from './client';
import type { ListOptions, PaginatedResponse, ResourceConfig } from './types';

export interface Template {
  id: string;
  name: string;
  description?: string;
  category?: string;
  tags?: string[];
  icon_url?: string;
  default_resources?: ResourceConfig;
  dockerfile?: string;
  setup_commands?: string[];
  is_official: boolean;
  is_public: boolean;
  version?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateRequest {
  name: string;
  description?: string;
  category?: string;
  tags?: string[];
  default_resources?: ResourceConfig;
  dockerfile: string;
  setup_commands?: string[];
  is_public?: boolean;
}

export interface UpdateTemplateRequest {
  name?: string;
  description?: string;
  category?: string;
  tags?: string[];
  default_resources?: ResourceConfig;
  dockerfile?: string;
  setup_commands?: string[];
  is_public?: boolean;
}

export interface TemplateListOptions extends ListOptions {
  category?: string;
  search?: string;
}

export class TemplatesService {
  constructor(private client: DevBoxClient) {}

  async list(options?: TemplateListOptions): Promise<PaginatedResponse<Template>> {
    const params = new URLSearchParams();
    if (options?.page) params.set('page', String(options.page));
    if (options?.pageSize) params.set('page_size', String(options.pageSize));
    if (options?.category) params.set('category', options.category);
    if (options?.search) params.set('search', options.search);
    
    const query = params.toString();
    return this.client.request('GET', `/templates${query ? `?${query}` : ''}`);
  }

  async get(id: string): Promise<Template> {
    return this.client.request('GET', `/templates/${id}`);
  }

  async create(request: CreateTemplateRequest): Promise<Template> {
    return this.client.request('POST', '/templates', request);
  }

  async update(id: string, request: UpdateTemplateRequest): Promise<Template> {
    return this.client.request('PUT', `/templates/${id}`, request);
  }

  async delete(id: string): Promise<void> {
    return this.client.request('DELETE', `/templates/${id}`);
  }
}
