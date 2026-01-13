/**
 * Cloud DevBox API Client
 */

import { APIError, DevBoxError } from './types';
import { EnvironmentsService } from './environments';
import { TemplatesService } from './templates';
import { DeploymentsService } from './deployments';
import { WebhooksService } from './webhooks';
import { APIKeysService } from './apikeys';

const DEFAULT_BASE_URL = 'https://api.clouddevbox.io/api/v1';
const DEFAULT_TIMEOUT = 30000;

export interface ClientOptions {
  baseUrl?: string;
  timeout?: number;
}

export class DevBoxClient {
  private baseUrl: string;
  private apiKey: string;
  private timeout: number;

  public readonly environments: EnvironmentsService;
  public readonly templates: TemplatesService;
  public readonly deployments: DeploymentsService;
  public readonly webhooks: WebhooksService;
  public readonly apiKeys: APIKeysService;

  constructor(apiKey: string, options: ClientOptions = {}) {
    this.apiKey = apiKey;
    this.baseUrl = options.baseUrl || DEFAULT_BASE_URL;
    this.timeout = options.timeout || DEFAULT_TIMEOUT;

    this.environments = new EnvironmentsService(this);
    this.templates = new TemplatesService(this);
    this.deployments = new DeploymentsService(this);
    this.webhooks = new WebhooksService(this);
    this.apiKeys = new APIKeysService(this);
  }

  async request<T>(
    method: string,
    path: string,
    body?: unknown
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers: {
          'X-API-Key': this.apiKey,
          'Content-Type': 'application/json',
          'Accept': 'application/json',
          'User-Agent': 'CloudDevBox-NodeJS-SDK/1.0',
        },
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const errorBody = await response.json() as APIError;
        throw new DevBoxError(errorBody);
      }

      if (response.status === 204) {
        return undefined as T;
      }

      return await response.json() as T;
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof DevBoxError) {
        throw error;
      }
      if (error instanceof Error && error.name === 'AbortError') {
        throw new Error('Request timeout');
      }
      throw error;
    }
  }
}
