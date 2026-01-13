/**
 * Billing Service
 * Handles billing operations including plans, subscriptions, quotas, and invoices
 */

import api from './api';

// Types
export interface Plan {
  id: string;
  name: string;
  display_name: string;
  description: string;
  price_monthly: number;
  price_yearly: number;
  currency: string;
  features: PlanFeatures;
  limits: PlanLimits;
  is_active: boolean;
  sort_order: number;
}

export interface PlanFeatures {
  environments: number | string;
  collaboration: boolean;
  support: string;
  custom_domains?: boolean;
  sso?: boolean;
  audit_logs?: boolean;
}

export interface PlanLimits {
  max_environments: number;
  cpu_hours_monthly: number;
  memory_gb_hours_monthly: number;
  storage_gb: number;
  build_minutes_monthly: number;
}

export interface Subscription {
  id: string;
  user_id: string;
  plan_id: string;
  plan?: Plan;
  status: SubscriptionStatus;
  billing_cycle: BillingCycle;
  current_period_start: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
  canceled_at?: string;
  trial_start?: string;
  trial_end?: string;
  created_at: string;
  updated_at: string;
}

export type SubscriptionStatus = 'active' | 'canceled' | 'past_due' | 'trialing' | 'paused';
export type BillingCycle = 'monthly' | 'yearly';

export interface Quota {
  id: string;
  user_id: string;
  resource_type: ResourceType;
  limit_value: number;
  used_value: number;
  reset_at?: string;
  created_at: string;
  updated_at: string;
}

export type ResourceType = 'cpu_hours' | 'memory_gb_hours' | 'storage_gb_days' | 'network_gb' | 'build_minutes';

export interface Invoice {
  id: string;
  user_id: string;
  subscription_id?: string;
  invoice_number: string;
  status: InvoiceStatus;
  currency: string;
  subtotal: number;
  tax: number;
  total: number;
  amount_paid: number;
  amount_due: number;
  due_date?: string;
  paid_at?: string;
  billing_period_start?: string;
  billing_period_end?: string;
  items?: InvoiceItem[];
  created_at: string;
  updated_at: string;
}

export type InvoiceStatus = 'draft' | 'open' | 'paid' | 'void' | 'uncollectible';

export interface InvoiceItem {
  id: string;
  invoice_id: string;
  description: string;
  quantity: number;
  unit_price: number;
  amount: number;
  created_at: string;
}

export interface UsageRecord {
  id: string;
  user_id: string;
  environment_id?: string;
  resource_type: ResourceType;
  quantity: number;
  unit: string;
  recorded_at: string;
  billing_period_start: string;
  billing_period_end: string;
}

export interface UsageSummary {
  resource_type: ResourceType;
  total_usage: number;
  unit: string;
  cost: number;
  period_start: string;
  period_end: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

// API Functions

/**
 * Get all available plans
 */
export async function getPlans(): Promise<Plan[]> {
  const response = await api.get<Plan[]>('/billing/plans');
  return response.data;
}

/**
 * Get a specific plan by ID
 */
export async function getPlan(planId: string): Promise<Plan> {
  const response = await api.get<Plan>(`/billing/plans/${planId}`);
  return response.data;
}

/**
 * Get current user's subscription
 */
export async function getSubscription(): Promise<Subscription | null> {
  const response = await api.get<Subscription | { subscription: null }>('/billing/subscription');
  if ('subscription' in response.data && response.data.subscription === null) {
    return null;
  }
  return response.data as Subscription;
}

/**
 * Create a new subscription
 */
export async function createSubscription(planId: string, billingCycle: BillingCycle = 'monthly'): Promise<Subscription> {
  const response = await api.post<Subscription>('/billing/subscription', {
    plan_id: planId,
    billing_cycle: billingCycle,
  });
  return response.data;
}

/**
 * Cancel subscription
 */
export async function cancelSubscription(immediate: boolean = false): Promise<{ message: string }> {
  const response = await api.post<{ message: string }>('/billing/subscription/cancel', { immediate });
  return response.data;
}

/**
 * Change subscription plan
 */
export async function changePlan(planId: string): Promise<Subscription> {
  const response = await api.post<Subscription>('/billing/subscription/change-plan', {
    plan_id: planId,
  });
  return response.data;
}

/**
 * Get user's quotas
 */
export async function getQuotas(): Promise<Quota[]> {
  const response = await api.get<Quota[]>('/billing/quotas');
  return response.data;
}

/**
 * Get user's invoices
 */
export async function getInvoices(page: number = 1, pageSize: number = 20): Promise<PaginatedResponse<Invoice>> {
  const response = await api.get<PaginatedResponse<Invoice>>('/billing/invoices', {
    params: { page, page_size: pageSize },
  });
  return response.data;
}

/**
 * Get a specific invoice
 */
export async function getInvoice(invoiceId: string): Promise<Invoice> {
  const response = await api.get<Invoice>(`/billing/invoices/${invoiceId}`);
  return response.data;
}

/**
 * Get usage summary for current billing period
 */
export async function getUsageSummary(): Promise<UsageSummary[]> {
  const response = await api.get<UsageSummary[]>('/billing/usage/summary');
  return response.data;
}

/**
 * Get detailed usage records
 */
export async function getUsageRecords(
  page: number = 1,
  pageSize: number = 50,
  resourceType?: ResourceType
): Promise<PaginatedResponse<UsageRecord>> {
  const response = await api.get<PaginatedResponse<UsageRecord>>('/billing/usage', {
    params: { page, page_size: pageSize, resource_type: resourceType },
  });
  return response.data;
}

/**
 * Download invoice as PDF
 */
export async function downloadInvoicePDF(invoiceId: string): Promise<Blob> {
  const response = await api.get(`/billing/invoices/${invoiceId}/pdf`, {
    responseType: 'blob',
  });
  return response.data;
}

// Helper functions

/**
 * Get resource type display name
 */
export function getResourceTypeLabel(resourceType: ResourceType): string {
  const labels: Record<ResourceType, string> = {
    cpu_hours: 'CPU Hours',
    memory_gb_hours: 'Memory (GB-Hours)',
    storage_gb_days: 'Storage (GB)',
    network_gb: 'Network (GB)',
    build_minutes: 'Build Minutes',
  };
  return labels[resourceType] || resourceType;
}

/**
 * Get resource type unit
 */
export function getResourceTypeUnit(resourceType: ResourceType): string {
  const units: Record<ResourceType, string> = {
    cpu_hours: 'hours',
    memory_gb_hours: 'GB-hours',
    storage_gb_days: 'GB',
    network_gb: 'GB',
    build_minutes: 'minutes',
  };
  return units[resourceType] || '';
}

/**
 * Calculate usage percentage
 */
export function calculateUsagePercent(used: number, limit: number): number {
  if (limit === -1) return 0; // Unlimited
  if (limit === 0) return 0;
  return Math.min((used / limit) * 100, 100);
}

/**
 * Check if quota is near limit (>80%)
 */
export function isQuotaNearLimit(used: number, limit: number): boolean {
  if (limit === -1) return false; // Unlimited
  return calculateUsagePercent(used, limit) >= 80;
}

/**
 * Check if quota is exceeded
 */
export function isQuotaExceeded(used: number, limit: number): boolean {
  if (limit === -1) return false; // Unlimited
  return used >= limit;
}

export default {
  getPlans,
  getPlan,
  getSubscription,
  createSubscription,
  cancelSubscription,
  changePlan,
  getQuotas,
  getInvoices,
  getInvoice,
  getUsageSummary,
  getUsageRecords,
  downloadInvoicePDF,
  getResourceTypeLabel,
  getResourceTypeUnit,
  calculateUsagePercent,
  isQuotaNearLimit,
  isQuotaExceeded,
};
