/**
 * Deployments Service
 */

import type { DevBoxClient } from './client';
import type { ListOptions, PaginatedResponse } from './types';

export interface Deployment {
  id: string;
  environment_id: string;
  version: string;
  status: 'pending' | 'building' | 'deploying' | 'succeeded' | 'failed' | 'rolled_back';
  image_tag?: string;
  commit_sha?: string;
  commit_message?: string;
  deployed_by?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface LogEntry {
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  message: string;
  stage?: 'build' | 'push' | 'deploy';
}

export interface DeploymentLogs {
  logs: LogEntry[];
}

export interface CreateDeploymentRequest {
  environment_id: string;
  branch?: string;
  commit_sha?: string;
  build_args?: Record<string, string>;
}

export interface DeploymentListOptions extends ListOptions {
  environmentId?: string;
  status?: string;
}

export class DeploymentsService {
  constructor(private client: DevBoxClient) {}

  async list(options?: DeploymentListOptions): Promise<PaginatedResponse<Deployment>> {
    const params = new URLSearchParams();
    if (options?.page) params.set('page', String(options.page));
    if (options?.pageSize) params.set('page_size', String(options.pageSize));
    if (options?.environmentId) params.set('environment_id', options.environmentId);
    if (options?.status) params.set('status', options.status);
    
    const query = params.toString();
    return this.client.request('GET', `/deployments${query ? `?${query}` : ''}`);
  }

  async get(id: string): Promise<Deployment> {
    return this.client.request('GET', `/deployments/${id}`);
  }

  async create(request: CreateDeploymentRequest): Promise<Deployment> {
    return this.client.request('POST', '/deployments', request);
  }

  async rollback(id: string): Promise<Deployment> {
    return this.client.request('POST', `/deployments/${id}/rollback`);
  }

  async getLogs(id: string): Promise<DeploymentLogs> {
    return this.client.request('GET', `/deployments/${id}/logs`);
  }
}
