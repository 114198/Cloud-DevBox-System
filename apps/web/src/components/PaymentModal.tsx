/**
 * Payment Modal Component
 * Handles payment flow for Alipay and WeChat Pay
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Modal,
  Button,
  Space,
  Typography,
  Spin,
  Result,
  QRCode,
} from 'antd';
import {
  AlipayCircleOutlined,
  WechatOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons';
import {
  createPayment,
  pollPaymentStatus,
  openAlipayPayment,
  Payment,
  PaymentProvider,
  CreatePaymentResponse,
} from '../services/payment';
import { useNumberFormat } from '../hooks/useLocale';

const { Title, Text } = Typography;

interface PaymentModalProps {
  visible: boolean;
  onClose: () => void;
  onSuccess?: (payment: Payment) => void;
  invoiceId: string;
  amount: number;
  currency?: string;
  description?: string;
}

type PaymentStep = 'select' | 'processing' | 'qrcode' | 'success' | 'failed';

export const PaymentModal: React.FC<PaymentModalProps> = ({
  visible,
  onClose,
  onSuccess,
  invoiceId,
  amount,
  currency = 'CNY',
  description,
}) => {
  const { t } = useTranslation();
  const { formatCurrency } = useNumberFormat();
  
  const [step, setStep] = useState<PaymentStep>('select');
  const [selectedProvider, setSelectedProvider] = useState<PaymentProvider | null>(null);
  const [loading, setLoading] = useState(false);
  const [paymentResponse, setPaymentResponse] = useState<CreatePaymentResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [, setPayment] = useState<Payment | null>(null);

  // Reset state when modal opens
  useEffect(() => {
    if (visible) {
      setStep('select');
      setSelectedProvider(null);
      setLoading(false);
      setPaymentResponse(null);
      setError(null);
      setPayment(null);
    }
  }, [visible]);

  // Handle payment creation
  const handleCreatePayment = useCallback(async (provider: PaymentProvider) => {
    setSelectedProvider(provider);
    setLoading(true);
    setError(null);

    try {
      const response = await createPayment({
        invoice_id: invoiceId,
        provider,
      });

      setPaymentResponse(response);
      setPayment(response.payment);

      if (provider === 'alipay' && response.payment_url) {
        // For Alipay, redirect to payment page
        setStep('processing');
        openAlipayPayment(response.payment_url);
        
        // Start polling for payment status
        pollPaymentStatus(response.payment.id, {
          interval: 3000,
          timeout: 300000,
          onStatusChange: (updatedPayment) => {
            setPayment(updatedPayment);
            if (updatedPayment.status === 'succeeded') {
              setStep('success');
              onSuccess?.(updatedPayment);
            } else if (updatedPayment.status === 'failed') {
              setStep('failed');
              setError(updatedPayment.failure_reason || 'Payment failed');
            }
          },
        }).catch((err) => {
          setStep('failed');
          setError(err.message);
        });
      } else if (provider === 'wechat' && response.qr_code_url) {
        // For WeChat, show QR code
        setStep('qrcode');
        
        // Start polling for payment status
        pollPaymentStatus(response.payment.id, {
          interval: 3000,
          timeout: 300000,
          onStatusChange: (updatedPayment) => {
            setPayment(updatedPayment);
            if (updatedPayment.status === 'succeeded') {
              setStep('success');
              onSuccess?.(updatedPayment);
            } else if (updatedPayment.status === 'failed') {
              setStep('failed');
              setError(updatedPayment.failure_reason || 'Payment failed');
            }
          },
        }).catch((err) => {
          setStep('failed');
          setError(err.message);
        });
      }
    } catch (err: any) {
      setError(err.message || 'Failed to create payment');
      setStep('failed');
    } finally {
      setLoading(false);
    }
  }, [invoiceId, onSuccess]);

  // Render payment provider selection
  const renderProviderSelection = () => (
    <div style={{ textAlign: 'center', padding: '24px 0' }}>
      <Title level={4}>{t('payment.selectMethod', 'Select Payment Method')}</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
        {description || t('payment.paymentFor', 'Payment for invoice')}
      </Text>
      
      <div style={{ 
        fontSize: 32, 
        fontWeight: 'bold', 
        marginBottom: 32,
        color: '#1890ff'
      }}>
        {formatCurrency(amount)}
      </div>

      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Button
          size="large"
          block
          icon={<AlipayCircleOutlined style={{ color: '#1677ff', fontSize: 24 }} />}
          onClick={() => handleCreatePayment('alipay')}
          loading={loading && selectedProvider === 'alipay'}
          disabled={loading}
          style={{ 
            height: 60, 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            gap: 12,
          }}
        >
          <span style={{ fontSize: 16 }}>{t('payment.alipay', 'Alipay')}</span>
        </Button>

        <Button
          size="large"
          block
          icon={<WechatOutlined style={{ color: '#07c160', fontSize: 24 }} />}
          onClick={() => handleCreatePayment('wechat')}
          loading={loading && selectedProvider === 'wechat'}
          disabled={loading}
          style={{ 
            height: 60, 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            gap: 12,
          }}
        >
          <span style={{ fontSize: 16 }}>{t('payment.wechat', 'WeChat Pay')}</span>
        </Button>
      </Space>
    </div>
  );

  // Render processing state (for Alipay redirect)
  const renderProcessing = () => (
    <div style={{ textAlign: 'center', padding: '48px 0' }}>
      <Spin indicator={<LoadingOutlined style={{ fontSize: 48 }} spin />} />
      <Title level={4} style={{ marginTop: 24 }}>
        {t('payment.processing', 'Processing Payment...')}
      </Title>
      <Text type="secondary">
        {t('payment.waitingForConfirmation', 'Please complete the payment in the opened window. This page will update automatically.')}
      </Text>
    </div>
  );

  // Render QR code (for WeChat)
  const renderQRCode = () => (
    <div style={{ textAlign: 'center', padding: '24px 0' }}>
      <Title level={4}>
        <WechatOutlined style={{ color: '#07c160', marginRight: 8 }} />
        {t('payment.scanQRCode', 'Scan QR Code to Pay')}
      </Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 24 }}>
        {t('payment.openWechat', 'Open WeChat and scan the QR code below')}
      </Text>

      <div style={{ 
        display: 'flex', 
        justifyContent: 'center', 
        marginBottom: 24,
        padding: 16,
        background: '#fff',
        borderRadius: 8,
        boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
      }}>
        {paymentResponse?.qr_code_url && (
          <QRCode 
            value={paymentResponse.qr_code_url} 
            size={200}
            icon="/wechat-pay-icon.png"
            iconSize={40}
          />
        )}
      </div>

      <div style={{ 
        fontSize: 24, 
        fontWeight: 'bold', 
        marginBottom: 16,
        color: '#07c160'
      }}>
        {formatCurrency(amount)}
      </div>

      <Space>
        <Spin size="small" />
        <Text type="secondary">
          {t('payment.waitingForPayment', 'Waiting for payment...')}
        </Text>
      </Space>
    </div>
  );

  // Render success state
  const renderSuccess = () => (
    <Result
      status="success"
      icon={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
      title={t('payment.success', 'Payment Successful!')}
      subTitle={t('payment.thankYou', 'Thank you for your payment.')}
      extra={[
        <Button type="primary" key="close" onClick={onClose}>
          {t('common.close', 'Close')}
        </Button>,
      ]}
    />
  );

  // Render failed state
  const renderFailed = () => (
    <Result
      status="error"
      icon={<CloseCircleOutlined style={{ color: '#ff4d4f' }} />}
      title={t('payment.failed', 'Payment Failed')}
      subTitle={error || t('payment.tryAgain', 'Please try again.')}
      extra={[
        <Button type="primary" key="retry" onClick={() => setStep('select')}>
          {t('payment.retry', 'Try Again')}
        </Button>,
        <Button key="close" onClick={onClose}>
          {t('common.close', 'Close')}
        </Button>,
      ]}
    />
  );

  // Render content based on step
  const renderContent = () => {
    switch (step) {
      case 'select':
        return renderProviderSelection();
      case 'processing':
        return renderProcessing();
      case 'qrcode':
        return renderQRCode();
      case 'success':
        return renderSuccess();
      case 'failed':
        return renderFailed();
      default:
        return renderProviderSelection();
    }
  };

  return (
    <Modal
      title={step === 'select' ? t('payment.title', 'Payment') : null}
      open={visible}
      onCancel={onClose}
      footer={null}
      width={480}
      centered
      closable={step !== 'processing'}
      maskClosable={step !== 'processing' && step !== 'qrcode'}
    >
      {renderContent()}
    </Modal>
  );
};

export default PaymentModal;
