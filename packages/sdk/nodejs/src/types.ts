/**
 * Common types for the Cloud DevBox SDK
 */

export interface Pagination {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: Pagination;
}

export interface ListOptions {
  page?: number;
  pageSize?: number;
}

export interface APIError {
  code: string;
  message: string;
  details?: Record<string, string>;
  request_id?: string;
  timestamp: string;
}

export class DevBoxError extends Error {
  code: string;
  details?: Record<string, string>;
  requestId?: string;

  constructor(error: APIError) {
    super(error.message);
    this.name = 'DevBoxError';
    this.code = error.code;
    this.details = error.details;
    this.requestId = error.request_id;
  }
}

export interface ResourceConfig {
  cpu: string;
  memory: string;
  storage: string;
}

export interface PortMapping {
  container_port: number;
  protocol: 'tcp' | 'udp';
  public: boolean;
}
