/**
 * Payment Service
 * Handles payment operations including Alipay and WeChat Pay integration
 */

import { api } from './api';

// Types
export interface Payment {
  id: string;
  user_id: string;
  invoice_id?: string;
  payment_method_id?: string;
  amount: number;
  currency: string;
  status: PaymentStatus;
  provider: PaymentProvider;
  failure_reason?: string;
  created_at: string;
  updated_at: string;
}

export type PaymentStatus = 'pending' | 'processing' | 'succeeded' | 'failed' | 'refunded';
export type PaymentProvider = 'alipay' | 'wechat';

export interface PaymentMethod {
  id: string;
  user_id: string;
  type: PaymentMethodType;
  provider: string;
  is_default: boolean;
  last_four?: string;
  brand?: string;
  exp_month?: number;
  exp_year?: number;
  created_at: string;
  updated_at: string;
}

export type PaymentMethodType = 'card' | 'alipay' | 'wechat' | 'bank_transfer';

export interface CreatePaymentRequest {
  invoice_id: string;
  provider: PaymentProvider;
}

export interface CreatePaymentResponse {
  payment: Payment;
  payment_url?: string;  // For Alipay redirect
  qr_code_url?: string;  // For WeChat QR code
  expires_at: string;
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
 * Create a new payment for an invoice
 */
export async function createPayment(request: CreatePaymentRequest): Promise<CreatePaymentResponse> {
  const response = await api.post<CreatePaymentResponse>('/api/v1/payments', request);
  return response.data;
}

/**
 * Get a payment by ID
 */
export async function getPayment(paymentId: string): Promise<Payment> {
  const response = await api.get<Payment>(`/api/v1/payments/${paymentId}`);
  return response.data;
}

/**
 * List user's payments
 */
export async function listPayments(page = 1, pageSize = 20): Promise<PaginatedResponse<Payment>> {
  const response = await api.get<PaginatedResponse<Payment>>('/api/v1/payments', {
    params: { page, page_size: pageSize },
  });
  return response.data;
}

/**
 * Query payment status from provider
 */
export async function queryPaymentStatus(paymentId: string): Promise<Payment> {
  const response = await api.get<Payment>(`/api/v1/payments/${paymentId}/status`);
  return response.data;
}

/**
 * Get user's saved payment methods
 */
export async function getPaymentMethods(): Promise<PaymentMethod[]> {
  const response = await api.get<PaymentMethod[]>('/api/v1/payments/methods');
  return response.data;
}

/**
 * Set a payment method as default
 */
export async function setDefaultPaymentMethod(methodId: string): Promise<void> {
  await api.post(`/api/v1/payments/methods/${methodId}/default`);
}

/**
 * Delete a payment method
 */
export async function deletePaymentMethod(methodId: string): Promise<void> {
  await api.delete(`/api/v1/payments/methods/${methodId}`);
}

/**
 * Poll payment status until it's completed or failed
 */
export async function pollPaymentStatus(
  paymentId: string,
  options: {
    interval?: number;
    timeout?: number;
    onStatusChange?: (payment: Payment) => void;
  } = {}
): Promise<Payment> {
  const { interval = 3000, timeout = 300000, onStatusChange } = options;
  const startTime = Date.now();

  return new Promise((resolve, reject) => {
    const poll = async () => {
      try {
        const payment = await queryPaymentStatus(paymentId);
        
        if (onStatusChange) {
          onStatusChange(payment);
        }

        if (payment.status === 'succeeded') {
          resolve(payment);
          return;
        }

        if (payment.status === 'failed') {
          reject(new Error(payment.failure_reason || 'Payment failed'));
          return;
        }

        if (Date.now() - startTime > timeout) {
          reject(new Error('Payment timeout'));
          return;
        }

        setTimeout(poll, interval);
      } catch (error) {
        reject(error);
      }
    };

    poll();
  });
}

/**
 * Open Alipay payment page
 */
export function openAlipayPayment(paymentUrl: string): void {
  window.location.href = paymentUrl;
}

/**
 * Generate QR code data URL from WeChat code URL
 * Note: In production, you would use a QR code library like qrcode
 */
export async function generateQRCodeDataURL(codeUrl: string): Promise<string> {
  // This is a placeholder - in production, use a QR code library
  // Example with qrcode library:
  // import QRCode from 'qrcode';
  // return QRCode.toDataURL(codeUrl);
  
  // For now, return a placeholder that indicates QR code generation is needed
  return `data:image/svg+xml,${encodeURIComponent(`
    <svg xmlns="http://www.w3.org/2000/svg" width="200" height="200">
      <rect width="200" height="200" fill="white"/>
      <text x="100" y="100" text-anchor="middle" font-size="12">QR Code: ${codeUrl.substring(0, 20)}...</text>
    </svg>
  `)}`;
}

export default {
  createPayment,
  getPayment,
  listPayments,
  queryPaymentStatus,
  getPaymentMethods,
  setDefaultPaymentMethod,
  deletePaymentMethod,
  pollPaymentStatus,
  openAlipayPayment,
  generateQRCodeDataURL,
};
