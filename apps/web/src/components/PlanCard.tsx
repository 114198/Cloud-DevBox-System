/**
 * PlanCard Component
 * Displays a subscription plan with features and pricing
 */

import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Typography,
  Space,
  Tag,
  Button,
  Divider,
} from 'antd';
import {
  CheckCircleOutlined,
  InfoCircleOutlined,
  CrownOutlined,
} from '@ant-design/icons';
import type { Plan, BillingCycle } from '../services/billing';

const { Title, Text } = Typography;

interface PlanCardProps {
  plan: Plan;
  billingCycle: BillingCycle;
  isCurrentPlan?: boolean;
  isSelected?: boolean;
  onSelect?: (planId: string) => void;
  loading?: boolean;
}

export const PlanCard: React.FC<PlanCardProps> = ({
  plan,
  billingCycle,
  isCurrentPlan = false,
  isSelected = false,
  onSelect,
  loading = false,
}: PlanCardProps) => {
  const { t } = useTranslation();

  const price = billingCycle === 'yearly'
    ? Math.round(plan.price_yearly / 12)
    : plan.price_monthly;

  const yearlyPrice = plan.price_yearly;
  const monthlySavings = billingCycle === 'yearly'
    ? plan.price_monthly * 12 - yearlyPrice
    : 0;

  const handleClick = () => {
    if (!isCurrentPlan && onSelect) {
      onSelect(plan.id);
    }
  };

  const getBorderColor = () => {
    if (isCurrentPlan) return '#1890ff';
    if (isSelected) return '#52c41a';
    return undefined;
  };

  return (
    <Card
      hoverable={!isCurrentPlan}
      onClick={handleClick}
      style={{
        borderColor: getBorderColor(),
        borderWidth: isCurrentPlan || isSelected ? 2 : 1,
        cursor: isCurrentPlan ? 'default' : 'pointer',
        height: '100%',
      }}
    >
      {isCurrentPlan && (
        <Tag color="blue" style={{ position: 'absolute', top: 12, right: 12 }}>
          <CrownOutlined style={{ marginRight: 4 }} />
          {t('billing.currentPlan', 'Current')}
        </Tag>
      )}

      {plan.name === 'pro' && !isCurrentPlan && (
        <Tag color="gold" style={{ position: 'absolute', top: 12, right: 12 }}>
          {t('billing.popular', 'Popular')}
        </Tag>
      )}

      <div style={{ textAlign: 'center', marginBottom: 16 }}>
        <Title level={4} style={{ marginBottom: 8 }}>
          {plan.display_name}
        </Title>
        <Title level={2} style={{ margin: '16px 0', color: '#1890ff' }}>
          ${price}
          <Text type="secondary" style={{ fontSize: 14 }}>/mo</Text>
        </Title>
        {billingCycle === 'yearly' && monthlySavings > 0 && (
          <Tag color="green">
            {t('billing.saveYearly', 'Save ${{amount}}/year', { amount: monthlySavings })}
          </Tag>
        )}
        <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
          {plan.description}
        </Text>
      </div>

      <Divider />

      <Space direction="vertical" style={{ width: '100%' }}>
        {/* Environments */}
        <div>
          <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
          {plan.limits.max_environments === -1
            ? t('billing.unlimited', 'Unlimited')
            : plan.limits.max_environments}{' '}
          {t('billing.environments', 'environments')}
        </div>

        {/* CPU Hours */}
        <div>
          <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
          {plan.limits.cpu_hours_monthly === -1
            ? t('billing.unlimited', 'Unlimited')
            : plan.limits.cpu_hours_monthly}{' '}
          {t('billing.cpuHours', 'CPU hours/mo')}
        </div>

        {/* Memory */}
        <div>
          <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
          {plan.limits.memory_gb_hours_monthly === -1
            ? t('billing.unlimited', 'Unlimited')
            : plan.limits.memory_gb_hours_monthly}{' '}
          {t('billing.memoryHours', 'GB-hours/mo')}
        </div>

        {/* Storage */}
        <div>
          <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
          {plan.limits.storage_gb === -1
            ? t('billing.unlimited', 'Unlimited')
            : `${plan.limits.storage_gb}GB`}{' '}
          {t('billing.storage', 'storage')}
        </div>

        {/* Build Minutes */}
        <div>
          <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
          {plan.limits.build_minutes_monthly === -1
            ? t('billing.unlimited', 'Unlimited')
            : plan.limits.build_minutes_monthly}{' '}
          {t('billing.buildMinutes', 'build minutes/mo')}
        </div>

        {/* Collaboration */}
        {plan.features.collaboration && (
          <div>
            <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
            {t('billing.collaboration', 'Real-time collaboration')}
          </div>
        )}

        {/* Custom Domains */}
        {plan.features.custom_domains && (
          <div>
            <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
            {t('billing.customDomains', 'Custom domains')}
          </div>
        )}

        {/* SSO */}
        {plan.features.sso && (
          <div>
            <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
            {t('billing.sso', 'SSO integration')}
          </div>
        )}

        {/* Audit Logs */}
        {plan.features.audit_logs && (
          <div>
            <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
            {t('billing.auditLogs', 'Audit logs')}
          </div>
        )}

        {/* Support Level */}
        <div>
          <InfoCircleOutlined style={{ color: '#1890ff', marginRight: 8 }} />
          {t(`billing.supportLevel.${plan.features.support}`, plan.features.support)}{' '}
          {t('billing.support', 'support')}
        </div>
      </Space>

      <Button
        type={isCurrentPlan ? 'default' : isSelected ? 'primary' : 'default'}
        block
        style={{ marginTop: 24 }}
        disabled={isCurrentPlan}
        loading={loading && isSelected}
      >
        {isCurrentPlan
          ? t('billing.currentPlan', 'Current Plan')
          : isSelected
          ? t('billing.selected', 'Selected')
          : t('billing.select', 'Select')}
      </Button>
    </Card>
  );
};

export default PlanCard;
