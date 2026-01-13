/**
 * Webhooks Service
 */

import type { DevBoxClient } from './client';
import type { ListOptions, PaginatedResponse } from './types';

export interface WebhookRetryConfig {
  max_retries: number;
  retry_delay_ms: number;
  backoff_factor: number;
  max_delay_ms: number;
}

export interface Webhook {
  id: string;
  name: string;
  description?: string;
  url: string;
  events: string[];
  is_active: boolean;
  headers?: Record<string, string>;
  retry_config?: WebhookRetryConfig;
  created_at: string;
  updated_at: string;
}

export interface WebhookDelivery {
  id: string;
  webhook_id: string;
  event_type: string;
  event_id: string;
  payload: string;
  response_code?: number;
  response_body?: string;
  status: 'pending' | 'success' | 'failed' | 'retrying';
  attempt_count: number;
  next_retry_at?: string;
  delivered_at?: string;
  error_message?: string;
  duration_ms?: number;
  created_at: string;
}

export interface CreateWebhookRequest {
  name: string;
  description?: string;
  url: string;
  events: string[];
  headers?: Record<string, string>;
  retry_config?: WebhookRetryConfig;
}

export class WebhooksService {
  constructor(private client: DevBoxClient) {}

  async list(options?: ListOptions): Promise<PaginatedResponse<Webhook>> {
    const params = new URLSearchParams();
    if (options?.page) params.set('page', String(options.page));
    if (options?.pageSize) params.set('page_size', String(options.pageSize));
    
    const query = params.toString();
    return this.client.request('GET', `/webhooks${query ? `?${query}` : ''}`);
  }

  async get(id: string): Promise<Webhook> {
    return this.client.request('GET', `/webhooks/${id}`);
  }

  async create(request: CreateWebhookRequest): Promise<Webhook> {
    return this.client.request('POST', '/webhooks', request);
  }

  async update(id: string, request: CreateWebhookRequest): Promise<Webhook> {
    return this.client.request('PUT', `/webhooks/${id}`, request);
  }

  async delete(id: string): Promise<void> {
    return this.client.request('DELETE', `/webhooks/${id}`);
  }

  async setActive(id: string, active: boolean): Promise<void> {
    return this.client.request('PUT', `/webhooks/${id}/active`, { active });
  }

  async regenerateSecret(id: string): Promise<string> {
    const response = await this.client.request<{ secret: string }>('POST', `/webhooks/${id}/secret`);
    return response.secret;
  }

  async test(id: string): Promise<void> {
    return this.client.request('POST', `/webhooks/${id}/test`);
  }

  async listDeliveries(webhookId: string, options?: ListOptions): Promise<PaginatedResponse<WebhookDelivery>> {
    const params = new URLSearchParams();
    if (options?.page) params.set('page', String(options.page));
    if (options?.pageSize) params.set('page_size', String(options.pageSize));
    
    const query = params.toString();
    return this.client.request('GET', `/webhooks/${webhookId}/deliveries${query ? `?${query}` : ''}`);
  }

  async retryDelivery(webhookId: string, deliveryId: string): Promise<void> {
    return this.client.request('POST', `/webhooks/${webhookId}/deliveries/${deliveryId}/retry`);
  }

  async getAvailableEvents(): Promise<string[]> {
    return this.client.request('GET', '/webhooks/events');
  }
}
