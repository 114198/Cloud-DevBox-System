// Preview service API client
import api from './api';

// Types
export interface Domain {
  id: string;
  environmentId: string;
  userId: string;
  type: 'Auto' | 'Custom';
  subdomain: string;
  customDomain?: string;
  fullDomain: string;
  port: number;
  status: 'Pending' | 'Active' | 'Failed' | 'Expired' | 'Deleting';
  message?: string;
  sslEnabled: boolean;
  sslCertId?: string;
  sslExpiresAt?: string;
  dnsRecordId?: string;
  dnsVerified: boolean;
  cnameTarget?: string;
  createdAt: string;
  updatedAt: string;
  verifiedAt?: string;
}

export interface ShareLink {
  id: string;
  domainId: string;
  environmentId: string;
  userId: string;
  token: string;
  url: string;
  password?: string;
  hasPassword: boolean;
  maxViews?: number;
  viewCount: number;
  expiresAt: string;
  duration: string;
  isActive: boolean;
  createdAt: string;
  lastAccessedAt?: string;
}

export interface HotReloadConfig {
  enabled: boolean;
  watchPaths: string[];
  ignorePaths: string[];
  debounceMs: number;
  notifyClients: boolean;
}

export interface PreviewSession {
  id: string;
  environmentId: string;
  domainId: string;
  clientId: string;
  userAgent?: string;
  ipAddress?: string;
  connectedAt: string;
  lastPingAt: string;
}

export interface PreviewStats {
  totalDomains: number;
  activeDomains: number;
  customDomains: number;
  totalShareLinks: number;
  activeShareLinks: number;
  totalViews: number;
}

export interface DomainValidationResult {
  valid: boolean;
  dnsVerified: boolean;
  sslValid: boolean;
  errors?: string[];
  warnings?: string[];
}

export interface CreateDomainRequest {
  environmentId: string;
  port: number;
  type?: 'Auto' | 'Custom';
  customDomain?: string;
}

export interface CreateShareLinkRequest {
  domainId: string;
  duration: '1h' | '6h' | '12h' | '24h' | '7d' | '14d' | '30d';
  password?: string;
  maxViews?: number;
}

export interface ValidateShareLinkRequest {
  token: string;
  password?: string;
}

// Domain API
export const domainApi = {
  // Create a new domain
  create: async (data: CreateDomainRequest): Promise<Domain> => {
    const response = await api.post('/preview/domains', data);
    return response.data;
  },

  // Get domain by ID
  get: async (id: string): Promise<Domain> => {
    const response = await api.get(`/preview/domains/${id}`);
    return response.data;
  },

  // List domains
  list: async (params?: {
    environmentId?: string;
    type?: string;
    status?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{ domains: Domain[]; total: number; page: number; pageSize: number }> => {
    const response = await api.get('/preview/domains', { params });
    return response.data;
  },

  // Update domain
  update: async (id: string, data: { port?: number; customDomain?: string }): Promise<Domain> => {
    const response = await api.put(`/preview/domains/${id}`, data);
    return response.data;
  },

  // Delete domain
  delete: async (id: string): Promise<void> => {
    await api.delete(`/preview/domains/${id}`);
  },

  // Verify custom domain
  verify: async (id: string): Promise<DomainValidationResult> => {
    const response = await api.post(`/preview/domains/${id}/verify`);
    return response.data;
  },

  // Get DNS instructions
  getDnsInstructions: async (id: string): Promise<{
    domain: string;
    cnameTarget: string;
    recordType: string;
    ttl: number;
    instructions: string[];
    verified: boolean;
    status: string;
  }> => {
    const response = await api.get(`/preview/domains/${id}/dns-instructions`);
    return response.data;
  },
};

// Share Link API
export const shareLinkApi = {
  // Create a new share link
  create: async (data: CreateShareLinkRequest): Promise<ShareLink> => {
    const response = await api.post('/preview/share-links', data);
    return response.data;
  },

  // Get share link by ID
  get: async (id: string): Promise<ShareLink> => {
    const response = await api.get(`/preview/share-links/${id}`);
    return response.data;
  },

  // List share links
  list: async (params?: {
    domainId?: string;
    environmentId?: string;
    activeOnly?: boolean;
    page?: number;
    pageSize?: number;
  }): Promise<{ shareLinks: ShareLink[]; total: number; page: number; pageSize: number }> => {
    const response = await api.get('/preview/share-links', { params });
    return response.data;
  },

  // Validate share link (public access)
  validate: async (data: ValidateShareLinkRequest): Promise<ShareLink> => {
    const response = await api.post('/preview/share-links/validate', data);
    return response.data;
  },

  // Revoke share link
  revoke: async (id: string): Promise<void> => {
    await api.post(`/preview/share-links/${id}/revoke`);
  },

  // Delete share link
  delete: async (id: string): Promise<void> => {
    await api.delete(`/preview/share-links/${id}`);
  },
};

// Hot Reload API
export const hotReloadApi = {
  // Get hot reload config
  getConfig: async (environmentId: string): Promise<HotReloadConfig> => {
    const response = await api.get(`/preview/environments/${environmentId}/hotreload`);
    return response.data;
  },

  // Set hot reload config
  setConfig: async (environmentId: string, config: HotReloadConfig): Promise<HotReloadConfig> => {
    const response = await api.put(`/preview/environments/${environmentId}/hotreload`, config);
    return response.data;
  },

  // Get active sessions
  getSessions: async (environmentId: string): Promise<PreviewSession[]> => {
    const response = await api.get(`/preview/environments/${environmentId}/sessions`);
    return response.data;
  },

  // Notify file change
  notifyFileChange: async (data: {
    environmentId: string;
    path: string;
    type: 'create' | 'modify' | 'delete';
  }): Promise<void> => {
    await api.post('/preview/file-change', data);
  },
};

// Preview Stats API
export const previewStatsApi = {
  // Get preview statistics
  get: async (): Promise<PreviewStats> => {
    const response = await api.get('/preview/stats');
    return response.data;
  },
};

// Helper functions
export const getDurationLabel = (duration: string): string => {
  const labels: Record<string, string> = {
    '1h': '1 hour',
    '6h': '6 hours',
    '12h': '12 hours',
    '24h': '24 hours',
    '7d': '7 days',
    '14d': '14 days',
    '30d': '30 days',
  };
  return labels[duration] || duration;
};

export const getStatusColor = (status: string): string => {
  const colors: Record<string, string> = {
    Pending: 'yellow',
    Active: 'green',
    Failed: 'red',
    Expired: 'gray',
    Deleting: 'orange',
  };
  return colors[status] || 'gray';
};

export const formatExpiresAt = (expiresAt: string): string => {
  const date = new Date(expiresAt);
  const now = new Date();
  const diff = date.getTime() - now.getTime();

  if (diff < 0) {
    return 'Expired';
  }

  const hours = Math.floor(diff / (1000 * 60 * 60));
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return `${days} day${days > 1 ? 's' : ''} left`;
  }
  if (hours > 0) {
    return `${hours} hour${hours > 1 ? 's' : ''} left`;
  }

  const minutes = Math.floor(diff / (1000 * 60));
  return `${minutes} minute${minutes > 1 ? 's' : ''} left`;
};
