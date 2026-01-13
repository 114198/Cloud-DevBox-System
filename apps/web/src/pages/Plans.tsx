/**
 * Plans Page
 * Dedicated page for plan comparison and upgrade/downgrade functionality
 */

import React, { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Row,
  Col,
  Button,
  Radio,
  Typography,
  Space,
  Tag,
  Divider,
  Table,
  Modal,
  Alert,
  message,
  Spin,
  Empty,
  Tooltip,
  Badge,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CrownOutlined,
  ArrowUpOutlined,
  ArrowDownOutlined,
  InfoCircleOutlined,
  ThunderboltOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  TeamOutlined,
  SafetyCertificateOutlined,
  CustomerServiceOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { RadioChangeEvent } from 'antd';
import {
  getPlans,
  getSubscription,
  changePlan,
  createSubscription,
  type Plan,
  type BillingCycle,
} from '../services/billing';

const { Title, Text, Paragraph } = Typography;

// Feature comparison data structure
interface FeatureRow {
  key: string;
  feature: string;
  category: string;
  tooltip?: string;
}

const featureRows: FeatureRow[] = [
  { key: 'environments', feature: 'billing.features.maxEnvironments', category: 'resources' },
  { key: 'cpu_hours', feature: 'billing.features.cpuHours', category: 'resources' },
  { key: 'memory', feature: 'billing.features.memoryHours', category: 'resources' },
  { key: 'storage', feature: 'billing.features.storage', category: 'resources' },
  { key: 'build_minutes', feature: 'billing.features.buildMinutes', category: 'resources' },
  { key: 'collaboration', feature: 'billing.features.collaboration', category: 'features', tooltip: 'billing.features.collaborationTooltip' },
  { key: 'custom_domains', feature: 'billing.features.customDomains', category: 'features', tooltip: 'billing.features.customDomainsTooltip' },
  { key: 'sso', feature: 'billing.features.sso', category: 'features', tooltip: 'billing.features.ssoTooltip' },
  { key: 'audit_logs', feature: 'billing.features.auditLogs', category: 'features', tooltip: 'billing.features.auditLogsTooltip' },
  { key: 'support', feature: 'billing.features.supportLevel', category: 'support' },
];

export const Plans: React.FC = () => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  // State
  const [billingCycle, setBillingCycle] = useState<BillingCycle>('monthly');
  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [confirmModalVisible, setConfirmModalVisible] = useState(false);
  const [viewMode, setViewMode] = useState<'cards' | 'table'>('cards');

  // Queries
  const { data: plans, isLoading: plansLoading } = useQuery({
    queryKey: ['plans'],
    queryFn: getPlans,
  });

  const { data: subscription, isLoading: subscriptionLoading } = useQuery({
    queryKey: ['subscription'],
    queryFn: getSubscription,
  });

  // Mutations
  const changePlanMutation = useMutation({
    mutationFn: (planId: string) => changePlan(planId),
    onSuccess: () => {
      message.success(t('billing.planChanged', 'Plan changed successfully!'));
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
      queryClient.invalidateQueries({ queryKey: ['quotas'] });
      setConfirmModalVisible(false);
      setSelectedPlanId(null);
    },
    onError: () => {
      message.error(t('billing.planChangeFailed', 'Failed to change plan'));
    },
  });

  const createSubscriptionMutation = useMutation({
    mutationFn: ({ planId, cycle }: { planId: string; cycle: BillingCycle }) =>
      createSubscription(planId, cycle),
    onSuccess: () => {
      message.success(t('billing.subscriptionCreated', 'Subscription created!'));
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
      queryClient.invalidateQueries({ queryKey: ['quotas'] });
      setConfirmModalVisible(false);
      setSelectedPlanId(null);
    },
    onError: () => {
      message.error(t('billing.subscriptionFailed', 'Failed to create subscription'));
    },
  });

  // Computed values
  const currentPlan = useMemo(() => {
    if (!subscription || !plans) return null;
    return plans.find((p: Plan) => p.id === subscription.plan_id);
  }, [subscription, plans]);

  const selectedPlan = useMemo(() => {
    if (!selectedPlanId || !plans) return null;
    return plans.find((p: Plan) => p.id === selectedPlanId);
  }, [selectedPlanId, plans]);

  const isUpgrade = useMemo(() => {
    if (!currentPlan || !selectedPlan) return true;
    return selectedPlan.price_monthly > currentPlan.price_monthly;
  }, [currentPlan, selectedPlan]);

  const sortedPlans = useMemo(() => {
    if (!plans) return [];
    return [...plans].sort((a, b) => a.sort_order - b.sort_order);
  }, [plans]);

  // Handlers
  const handleBillingCycleChange = (e: RadioChangeEvent) => {
    setBillingCycle(e.target.value);
  };

  const handleSelectPlan = (planId: string) => {
    if (subscription?.plan_id === planId) return;
    setSelectedPlanId(planId);
    setConfirmModalVisible(true);
  };

  const handleConfirmChange = () => {
    if (!selectedPlanId) return;

    if (subscription) {
      changePlanMutation.mutate(selectedPlanId);
    } else {
      createSubscriptionMutation.mutate({ planId: selectedPlanId, cycle: billingCycle });
    }
  };

  const handleCancelModal = () => {
    setConfirmModalVisible(false);
    setSelectedPlanId(null);
  };

  // Get feature value for a plan
  const getFeatureValue = (plan: Plan, featureKey: string): React.ReactNode => {
    switch (featureKey) {
      case 'environments':
        return plan.limits.max_environments === -1
          ? t('billing.unlimited', 'Unlimited')
          : plan.limits.max_environments;
      case 'cpu_hours':
        return plan.limits.cpu_hours_monthly === -1
          ? t('billing.unlimited', 'Unlimited')
          : `${plan.limits.cpu_hours_monthly} ${t('billing.hoursPerMonth', 'hrs/mo')}`;
      case 'memory':
        return plan.limits.memory_gb_hours_monthly === -1
          ? t('billing.unlimited', 'Unlimited')
          : `${plan.limits.memory_gb_hours_monthly} ${t('billing.gbHoursPerMonth', 'GB-hrs/mo')}`;
      case 'storage':
        return plan.limits.storage_gb === -1
          ? t('billing.unlimited', 'Unlimited')
          : `${plan.limits.storage_gb} GB`;
      case 'build_minutes':
        return plan.limits.build_minutes_monthly === -1
          ? t('billing.unlimited', 'Unlimited')
          : `${plan.limits.build_minutes_monthly} ${t('billing.minsPerMonth', 'mins/mo')}`;
      case 'collaboration':
        return plan.features.collaboration ? (
          <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 18 }} />
        ) : (
          <CloseCircleOutlined style={{ color: '#d9d9d9', fontSize: 18 }} />
        );
      case 'custom_domains':
        return plan.features.custom_domains ? (
          <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 18 }} />
        ) : (
          <CloseCircleOutlined style={{ color: '#d9d9d9', fontSize: 18 }} />
        );
      case 'sso':
        return plan.features.sso ? (
          <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 18 }} />
        ) : (
          <CloseCircleOutlined style={{ color: '#d9d9d9', fontSize: 18 }} />
        );
      case 'audit_logs':
        return plan.features.audit_logs ? (
          <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 18 }} />
        ) : (
          <CloseCircleOutlined style={{ color: '#d9d9d9', fontSize: 18 }} />
        );
      case 'support':
        return t(`billing.supportLevel.${plan.features.support}`, plan.features.support);
      default:
        return '-';
    }
  };

  // Render plan card
  const renderPlanCard = (plan: Plan) => {
    const isCurrentPlan = subscription?.plan_id === plan.id;
    const price = billingCycle === 'yearly'
      ? Math.round(plan.price_yearly / 12)
      : plan.price_monthly;
    const yearlyTotal = plan.price_yearly;
    const monthlySavings = billingCycle === 'yearly'
      ? plan.price_monthly * 12 - yearlyTotal
      : 0;
    const isPopular = plan.name === 'pro';

    return (
      <Col xs={24} sm={12} lg={6} key={plan.id}>
        <Badge.Ribbon
          text={isPopular ? t('billing.popular', 'Popular') : ''}
          color="gold"
          style={{ display: isPopular ? 'block' : 'none' }}
        >
          <Card
            hoverable={!isCurrentPlan}
            onClick={() => !isCurrentPlan && handleSelectPlan(plan.id)}
            style={{
              height: '100%',
              borderColor: isCurrentPlan ? '#1890ff' : undefined,
              borderWidth: isCurrentPlan ? 2 : 1,
              cursor: isCurrentPlan ? 'default' : 'pointer',
            }}
          >
            {isCurrentPlan && (
              <Tag color="blue" style={{ position: 'absolute', top: 12, right: 12 }}>
                <CrownOutlined style={{ marginRight: 4 }} />
                {t('billing.currentPlan', 'Current')}
              </Tag>
            )}

            <div style={{ textAlign: 'center', marginBottom: 16, paddingTop: isCurrentPlan ? 24 : 0 }}>
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
              <Paragraph type="secondary" style={{ marginTop: 8, minHeight: 44 }}>
                {plan.description}
              </Paragraph>
            </div>

            <Divider />

            <Space direction="vertical" style={{ width: '100%' }} size="small">
              <div>
                <ThunderboltOutlined style={{ color: '#1890ff', marginRight: 8 }} />
                {plan.limits.max_environments === -1
                  ? t('billing.unlimited', 'Unlimited')
                  : plan.limits.max_environments}{' '}
                {t('billing.environments', 'environments')}
              </div>
              <div>
                <CloudServerOutlined style={{ color: '#1890ff', marginRight: 8 }} />
                {plan.limits.cpu_hours_monthly === -1
                  ? t('billing.unlimited', 'Unlimited')
                  : plan.limits.cpu_hours_monthly}{' '}
                {t('billing.cpuHours', 'CPU hours/mo')}
              </div>
              <div>
                <DatabaseOutlined style={{ color: '#1890ff', marginRight: 8 }} />
                {plan.limits.storage_gb === -1
                  ? t('billing.unlimited', 'Unlimited')
                  : `${plan.limits.storage_gb}GB`}{' '}
                {t('billing.storage', 'storage')}
              </div>
              {plan.features.collaboration && (
                <div>
                  <TeamOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                  {t('billing.collaboration', 'Real-time collaboration')}
                </div>
              )}
              {plan.features.custom_domains && (
                <div>
                  <SafetyCertificateOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                  {t('billing.customDomains', 'Custom domains')}
                </div>
              )}
              <div>
                <CustomerServiceOutlined style={{ color: '#1890ff', marginRight: 8 }} />
                {t(`billing.supportLevel.${plan.features.support}`, plan.features.support)}{' '}
                {t('billing.support', 'support')}
              </div>
            </Space>

            <Button
              type={isCurrentPlan ? 'default' : 'primary'}
              block
              style={{ marginTop: 24 }}
              disabled={isCurrentPlan}
            >
              {isCurrentPlan
                ? t('billing.currentPlan', 'Current Plan')
                : currentPlan && plan.price_monthly < currentPlan.price_monthly
                ? t('billing.downgrade', 'Downgrade')
                : t('billing.upgrade', 'Upgrade')}
            </Button>
          </Card>
        </Badge.Ribbon>
      </Col>
    );
  };

  // Render comparison table
  const renderComparisonTable = () => {
    const columns = [
      {
        title: t('billing.feature', 'Feature'),
        dataIndex: 'feature',
        key: 'feature',
        fixed: 'left' as const,
        width: 200,
        render: (text: string, record: FeatureRow) => (
          <Space>
            <span>{t(text, text)}</span>
            {record.tooltip && (
              <Tooltip title={t(record.tooltip, '')}>
                <InfoCircleOutlined style={{ color: '#999' }} />
              </Tooltip>
            )}
          </Space>
        ),
      },
      ...sortedPlans.map((plan: Plan) => ({
        title: (
          <div style={{ textAlign: 'center' }}>
            <div>{plan.display_name}</div>
            {subscription?.plan_id === plan.id && (
              <Tag color="blue" size="small">{t('billing.current', 'Current')}</Tag>
            )}
          </div>
        ),
        dataIndex: plan.id,
        key: plan.id,
        width: 150,
        align: 'center' as const,
        render: (_: unknown, record: FeatureRow) => getFeatureValue(plan, record.key),
      })),
    ];

    return (
      <Table
        dataSource={featureRows}
        columns={columns}
        pagination={false}
        scroll={{ x: 'max-content' }}
        rowKey="key"
        bordered
        size="middle"
      />
    );
  };

  // Render price comparison
  const renderPriceComparison = () => {
    if (!selectedPlan || !currentPlan) return null;

    const currentPrice = billingCycle === 'yearly'
      ? currentPlan.price_yearly
      : currentPlan.price_monthly;
    const newPrice = billingCycle === 'yearly'
      ? selectedPlan.price_yearly
      : selectedPlan.price_monthly;
    const priceDiff = newPrice - currentPrice;

    return (
      <div style={{ marginTop: 16 }}>
        <Row gutter={16}>
          <Col span={8}>
            <Card size="small">
              <Text type="secondary">{t('billing.currentPrice', 'Current Price')}</Text>
              <Title level={4} style={{ margin: '8px 0 0' }}>
                ${currentPrice}/{billingCycle === 'yearly' ? t('billing.year', 'yr') : t('billing.month', 'mo')}
              </Title>
            </Card>
          </Col>
          <Col span={8}>
            <Card size="small">
              <Text type="secondary">{t('billing.newPrice', 'New Price')}</Text>
              <Title level={4} style={{ margin: '8px 0 0', color: '#1890ff' }}>
                ${newPrice}/{billingCycle === 'yearly' ? t('billing.year', 'yr') : t('billing.month', 'mo')}
              </Title>
            </Card>
          </Col>
          <Col span={8}>
            <Card size="small">
              <Text type="secondary">{t('billing.difference', 'Difference')}</Text>
              <Title
                level={4}
                style={{
                  margin: '8px 0 0',
                  color: priceDiff > 0 ? '#ff4d4f' : '#52c41a',
                }}
              >
                {priceDiff > 0 ? '+' : ''}{priceDiff === 0 ? '$0' : `$${priceDiff}`}
                /{billingCycle === 'yearly' ? t('billing.year', 'yr') : t('billing.month', 'mo')}
              </Title>
            </Card>
          </Col>
        </Row>
      </div>
    );
  };

  // Loading state
  if (plansLoading || subscriptionLoading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  // Empty state
  if (!plans || plans.length === 0) {
    return (
      <div style={{ padding: 24 }}>
        <Empty
          description={t('billing.noPlans', 'No plans available')}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={2}>{t('billing.comparePlans', 'Compare Plans')}</Title>
        <Paragraph type="secondary">
          {t('billing.plansDescription', 'Choose the plan that best fits your needs. Upgrade or downgrade anytime.')}
        </Paragraph>
      </div>

      {/* Billing cycle toggle */}
      <div style={{ marginBottom: 24, textAlign: 'center' }}>
        <Radio.Group
          value={billingCycle}
          onChange={handleBillingCycleChange}
          buttonStyle="solid"
          size="large"
        >
          <Radio.Button value="monthly">{t('billing.monthly', 'Monthly')}</Radio.Button>
          <Radio.Button value="yearly">
            {t('billing.yearly', 'Yearly')}
            <Tag color="green" style={{ marginLeft: 8 }}>{t('billing.save17', 'Save 17%')}</Tag>
          </Radio.Button>
        </Radio.Group>
      </div>

      {/* View mode toggle */}
      <div style={{ marginBottom: 24, textAlign: 'right' }}>
        <Radio.Group
          value={viewMode}
          onChange={(e: RadioChangeEvent) => setViewMode(e.target.value)}
          optionType="button"
          buttonStyle="solid"
        >
          <Radio.Button value="cards">{t('billing.cardView', 'Cards')}</Radio.Button>
          <Radio.Button value="table">{t('billing.tableView', 'Table')}</Radio.Button>
        </Radio.Group>
      </div>

      {/* Plan cards or comparison table */}
      {viewMode === 'cards' ? (
        <Row gutter={[24, 24]}>
          {sortedPlans.map(renderPlanCard)}
        </Row>
      ) : (
        <Card>
          {renderComparisonTable()}
        </Card>
      )}

      {/* Plan selection buttons for table view */}
      {viewMode === 'table' && (
        <Row gutter={24} style={{ marginTop: 24 }}>
          {sortedPlans.map((plan: Plan) => {
            const isCurrentPlan = subscription?.plan_id === plan.id;
            const price = billingCycle === 'yearly'
              ? Math.round(plan.price_yearly / 12)
              : plan.price_monthly;

            return (
              <Col xs={24} sm={12} lg={6} key={plan.id}>
                <Card size="small" style={{ textAlign: 'center' }}>
                  <Title level={4} style={{ marginBottom: 8 }}>{plan.display_name}</Title>
                  <Title level={3} style={{ color: '#1890ff', margin: '8px 0' }}>
                    ${price}<Text type="secondary" style={{ fontSize: 14 }}>/mo</Text>
                  </Title>
                  <Button
                    type={isCurrentPlan ? 'default' : 'primary'}
                    block
                    disabled={isCurrentPlan}
                    onClick={() => handleSelectPlan(plan.id)}
                  >
                    {isCurrentPlan
                      ? t('billing.currentPlan', 'Current Plan')
                      : currentPlan && plan.price_monthly < currentPlan.price_monthly
                      ? t('billing.downgrade', 'Downgrade')
                      : t('billing.upgrade', 'Upgrade')}
                  </Button>
                </Card>
              </Col>
            );
          })}
        </Row>
      )}

      {/* Confirmation Modal */}
      <Modal
        title={
          <Space>
            {isUpgrade ? (
              <ArrowUpOutlined style={{ color: '#52c41a' }} />
            ) : (
              <ArrowDownOutlined style={{ color: '#faad14' }} />
            )}
            {isUpgrade
              ? t('billing.confirmUpgrade', 'Confirm Upgrade')
              : t('billing.confirmDowngrade', 'Confirm Downgrade')}
          </Space>
        }
        open={confirmModalVisible}
        onCancel={handleCancelModal}
        footer={[
          <Button key="cancel" onClick={handleCancelModal}>
            {t('common.cancel', 'Cancel')}
          </Button>,
          <Button
            key="confirm"
            type="primary"
            loading={changePlanMutation.isPending || createSubscriptionMutation.isPending}
            onClick={handleConfirmChange}
          >
            {t('billing.confirmChange', 'Confirm Change')}
          </Button>,
        ]}
        width={600}
      >
        {selectedPlan && (
          <>
            <Alert
              type={isUpgrade ? 'info' : 'warning'}
              message={
                isUpgrade
                  ? t('billing.upgradeInfo', 'You are upgrading to {{plan}}', { plan: selectedPlan.display_name })
                  : t('billing.downgradeInfo', 'You are downgrading to {{plan}}', { plan: selectedPlan.display_name })
              }
              description={
                isUpgrade
                  ? t('billing.upgradeDescription', 'Your new plan will take effect immediately. You will be charged the prorated difference.')
                  : t('billing.downgradeDescription', 'Your new plan will take effect at the end of your current billing period. Some features may become unavailable.')
              }
              showIcon
              style={{ marginBottom: 16 }}
            />

            {currentPlan && renderPriceComparison()}

            <Divider />

            <Title level={5}>{t('billing.planDetails', 'Plan Details')}</Title>
            <Space direction="vertical" style={{ width: '100%' }}>
              <div>
                <Text strong>{t('billing.environments', 'Environments')}: </Text>
                <Text>
                  {selectedPlan.limits.max_environments === -1
                    ? t('billing.unlimited', 'Unlimited')
                    : selectedPlan.limits.max_environments}
                </Text>
              </div>
              <div>
                <Text strong>{t('billing.cpuHours', 'CPU Hours')}: </Text>
                <Text>
                  {selectedPlan.limits.cpu_hours_monthly === -1
                    ? t('billing.unlimited', 'Unlimited')
                    : `${selectedPlan.limits.cpu_hours_monthly}/mo`}
                </Text>
              </div>
              <div>
                <Text strong>{t('billing.storage', 'Storage')}: </Text>
                <Text>
                  {selectedPlan.limits.storage_gb === -1
                    ? t('billing.unlimited', 'Unlimited')
                    : `${selectedPlan.limits.storage_gb}GB`}
                </Text>
              </div>
            </Space>
          </>
        )}
      </Modal>
    </div>
  );
};

export default Plans;
