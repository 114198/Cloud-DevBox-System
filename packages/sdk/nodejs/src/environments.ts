/**
 * Environments Service
 */

import type { DevBoxClient } from './client';
import type { ListOptions, PaginatedResponse, ResourceConfig, PortMapping } from './types';

export interface Environment {
  id: string;
  name: string;
  description?: string;
  template_id: string;
  status: 'creating' | 'running' | 'stopped' | 'failed' | 'suspended' | 'archived';
  resources?: ResourceConfig;
  env_vars?: Record<string, string>;
  ports?: PortMapping[];
  preview_url?: string;
  ssh_host?: string;
  ssh_port?: number;
  created_at: string;
  updated_at: string;
  last_active_at?: string;
}

export interface SSHConfig {
  host: string;
  port: number;
  username: string;
  private_key: string;
  config_snippet: string;
}

export interface MetricPoint {
  timestamp: string;
  value: number;
}

export interface EnvironmentMetrics {
  cpu_usage: MetricPoint[];
  memory_usage: MetricPoint[];
  storage_usage: MetricPoint[];
  network_rx: MetricPoint[];
  network_tx: MetricPoint[];
}

export interface CreateEnvironmentRequest {
  name: string;
  description?: string;
  template_id: string;
  resources?: ResourceConfig;
  env_vars?: Record<string, string>;
  ports?: PortMapping[];
  git_repo_url?: string;
}

export interface UpdateEnvironmentRequest {
  name?: string;
  description?: string;
  resources?: ResourceConfig;
  env_vars?: Record<string, string>;
}

export interface EnvironmentListOptions extends ListOptions {
  status?: string;
  templateId?: string;
}

export class EnvironmentsService {
  constructor(private client: DevBoxClient) {}

  async list(options?: EnvironmentListOptions): Promise<PaginatedResponse<Environment>> {
    const params = new URLSearchParams();
    if (options?.page) params.set('page', String(options.page));
    if (options?.pageSize) params.set('page_size', String(options.pageSize));
    if (options?.status) params.set('status', options.status);
    if (options?.templateId) params.set('template_id', options.templateId);
    
    const query = params.toString();
    return this.client.request('GET', `/environments${query ? `?${query}` : ''}`);
  }

  async get(id: string): Promise<Environment> {
    return this.client.request('GET', `/environments/${id}`);
  }

  async create(request: CreateEnvironmentRequest): Promise<Environment> {
    return this.client.request('POST', '/environments', request);
  }

  async update(id: string, request: UpdateEnvironmentRequest): Promise<Environment> {
    return this.client.request('PUT', `/environments/${id}`, request);
  }

  async delete(id: string): Promise<void> {
    return this.client.request('DELETE', `/environments/${id}`);
  }

  async start(id: string): Promise<Environment> {
    return this.client.request('POST', `/environments/${id}/start`);
  }

  async stop(id: string): Promise<Environment> {
    return this.client.request('POST', `/environments/${id}/stop`);
  }

  async restart(id: string): Promise<Environment> {
    return this.client.request('POST', `/environments/${id}/restart`);
  }

  async getSSHConfig(id: string): Promise<SSHConfig> {
    return this.client.request('GET', `/environments/${id}/ssh-config`);
  }

  async getMetrics(id: string, startTime?: Date, endTime?: Date): Promise<EnvironmentMetrics> {
    const params = new URLSearchParams();
    if (startTime) params.set('start_time', startTime.toISOString());
    if (endTime) params.set('end_time', endTime.toISOString());
    
    const query = params.toString();
    return this.client.request('GET', `/environments/${id}/metrics${query ? `?${query}` : ''}`);
  }
}
