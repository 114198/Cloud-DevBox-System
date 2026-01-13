/**
 * API Keys Service
 */

import type { DevBoxClient } from './client';
import type { ListOptions, PaginatedResponse } from './types';

export interface APIKey {
  id: string;
  name: string;
  description?: string;
  key_prefix: string;
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  last_used_ip?: string;
  is_active: boolean;
  created_at: string;
}

export interface CreateAPIKeyRequest {
  name: string;
  description?: string;
  scopes: string[];
  expires_at?: string;
}

export interface CreateAPIKeyResponse {
  api_key: APIKey;
  plain_key: string;
}

export interface UpdateAPIKeyRequest {
  name?: string;
  description?: string;
  scopes?: string[];
}

export class APIKeysService {
  constructor(private client: DevBoxClient) {}

  async list(options?: ListOptions): Promise<PaginatedResponse<APIKey>> {
    const params = new URLSearchParams();
    if (options?.page) params.set('page', String(options.page));
    if (options?.pageSize) params.set('page_size', String(options.pageSize));
    
    const query = params.toString();
    return this.client.request('GET', `/api-keys${query ? `?${query}` : ''}`);
  }

  async get(id: string): Promise<APIKey> {
    return this.client.request('GET', `/api-keys/${id}`);
  }

  async create(request: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
    return this.client.request('POST', '/api-keys', request);
  }

  async update(id: string, request: UpdateAPIKeyRequest): Promise<APIKey> {
    return this.client.request('PUT', `/api-keys/${id}`, request);
  }

  async delete(id: string): Promise<void> {
    return this.client.request('DELETE', `/api-keys/${id}`);
  }

  async revoke(id: string): Promise<void> {
    return this.client.request('POST', `/api-keys/${id}/revoke`);
  }

  async getAvailableScopes(): Promise<string[]> {
    return this.client.request('GET', '/api-keys/scopes');
  }
}
