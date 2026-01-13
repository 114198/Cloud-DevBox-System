/**
 * Billing Page
 * Includes usage reports, quota display, and plan management
 */

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  Row,
  Col,
  Button,
  Progress,
  Table,
  Tag,
  Space,
  Typography,
  Statistic,
  Modal,
  Radio,
  message,
  Tabs,
  Alert,
  Tooltip,
  Divider,
  Empty,
  Spin,
} from 'antd';
import {
  CrownOutlined,
  CheckCircleOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  RiseOutlined,
  ThunderboltOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  ClockCircleOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useDateFormat, useNumberFormat } from '../hooks/useLocale';
import PaymentModal from '../components/PaymentModal';
import type { RadioChangeEvent } from 'antd';
import {
  getPlans,
  getSubscription,
  getQuotas,
  getInvoices,
  getUsageSummary,
  changePlan,
  cancelSubscription,
  createSubscription,
  downloadInvoicePDF,
  getResourceTypeLabel,
  calculateUsagePercent,
  isQuotaNearLimit,
  isQuotaExceeded,
  type Plan,
  type Quota,
  type Invoice,
  type UsageSummary,
  type BillingCycle,
  type ResourceType,
} from '../services/billing';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

// Resource type icons
const resourceIcons: Record<ResourceType, React.ReactNode> = {
  cpu_hours: <ThunderboltOutlined />,
  memory_gb_hours: <CloudServerOutlined />,
  storage_gb_days: <DatabaseOutlined />,
  network_gb: <RiseOutlined />,
  build_minutes: <ClockCircleOutlined />,
};

export const Billing: React.FC = () => {
  const { t } = useTranslation();
  const { formatDate } = useDateFormat();
  const { formatCurrency } = useNumberFormat();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  // State
  const [activeTab, setActiveTab] = useState('overview');
  const [upgradeModalVisible, setUpgradeModalVisible] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<string | null>(null);
  const [billingCycle, setBillingCycle] = useState<BillingCycle>('monthly');
  const [paymentModalVisible, setPaymentModalVisible] = useState(false);
  const [selectedInvoice, setSelectedInvoice] = useState<Invoice | null>(null);
  const [cancelModalVisible, setCancelModalVisible] = useState(false);

  // Queries
  const { data: plans, isLoading: plansLoading } = useQuery({
    queryKey: ['plans'],
    queryFn: getPlans,
  });

  const { data: subscription, isLoading: subscriptionLoading } = useQuery({
    queryKey: ['subscription'],
    queryFn: getSubscription,
  });

  const { data: quotas, isLoading: quotasLoading } = useQuery({
    queryKey: ['quotas'],
    queryFn: getQuotas,
  });

  const { data: invoicesData, isLoading: invoicesLoading } = useQuery({
    queryKey: ['invoices'],
    queryFn: () => getInvoices(1, 20),
  });

  const { data: usageSummary, isLoading: usageLoading } = useQuery({
    queryKey: ['usageSummary'],
    queryFn: getUsageSummary,
  });

  // Mutations
  const changePlanMutation = useMutation({
    mutationFn: (planId: string) => changePlan(planId),
    onSuccess: () => {
      message.success(t('billing.planChanged', 'Plan changed successfully!'));
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
      queryClient.invalidateQueries({ queryKey: ['quotas'] });
      setUpgradeModalVisible(false);
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
      setUpgradeModalVisible(false);
    },
    onError: () => {
      message.error(t('billing.subscriptionFailed', 'Failed to create subscription'));
    },
  });

  const cancelSubscriptionMutation = useMutation({
    mutationFn: (immediate: boolean) => cancelSubscription(immediate),
    onSuccess: (data: { message: string }) => {
      message.success(data.message);
      queryClient.invalidateQueries({ queryKey: ['subscription'] });
      setCancelModalVisible(false);
    },
    onError: () => {
      message.error(t('billing.cancelFailed', 'Failed to cancel subscription'));
    },
  });

  // Handlers
  const handlePayInvoice = (invoice: Invoice) => {
    setSelectedInvoice(invoice);
    setPaymentModalVisible(true);
  };

  const handlePaymentSuccess = () => {
    message.success(t('payment.success', 'Payment successful!'));
    setPaymentModalVisible(false);
    setSelectedInvoice(null);
    queryClient.invalidateQueries({ queryKey: ['invoices'] });
  };

  const handleDownloadInvoice = async (invoiceId: string, invoiceNumber: string) => {
    try {
      const blob = await downloadInvoicePDF(invoiceId);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${invoiceNumber}.pdf`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch {
      message.error(t('billing.downloadFailed', 'Failed to download invoice'));
    }
  };

  const handleSelectPlan = (planId: string) => {
    setSelectedPlan(planId);
  };

  const handleConfirmPlanChange = () => {
    if (!selectedPlan) return;
    
    if (subscription) {
      changePlanMutation.mutate(selectedPlan);
    } else {
      createSubscriptionMutation.mutate({ planId: selectedPlan, cycle: billingCycle });
    }
  };

  const handleBillingCycleChange = (e: RadioChangeEvent) => {
    setBillingCycle(e.target.value);
  };

  // Invoice table columns
  const invoiceColumns = [
    {
      title: t('billing.invoiceNumber', 'Invoice #'),
      dataIndex: 'invoice_number',
      key: 'invoice_number',
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const colors: Record<string, string> = {
          paid: 'green',
          open: 'blue',
          draft: 'default',
          void: 'red',
          uncollectible: 'orange',
        };
        return (
          <Tag color={colors[status] || 'default'}>
            {t(`billing.invoiceStatus.${status}`, status.toUpperCase())}
          </Tag>
        );
      },
    },
    {
      title: t('billing.amount', 'Amount'),
      dataIndex: 'total',
      key: 'total',
      render: (total: number) => formatCurrency(total),
    },
    {
      title: t('billing.billingPeriod', 'Billing Period'),
      key: 'period',
      render: (_: unknown, record: Invoice) => {
        if (record.billing_period_start && record.billing_period_end) {
          return `${formatDate(record.billing_period_start)} - ${formatDate(record.billing_period_end)}`;
        }
        return '-';
      },
    },
    {
      title: t('common.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => formatDate(date),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      render: (_: unknown, record: Invoice) => (
        <Space>
          {record.status === 'open' && (
            <Button type="primary" size="small" onClick={() => handlePayInvoice(record)}>
              {t('billing.pay', 'Pay')}
            </Button>
          )}
          <Button
            type="link"
            icon={<DownloadOutlined />}
            size="small"
            onClick={() => handleDownloadInvoice(record.id, record.invoice_number)}
          >
            PDF
          </Button>
        </Space>
      ),
    },
  ];

  // Render current plan card
  const renderCurrentPlan = () => {
    if (subscriptionLoading) {
      return <Spin />;
    }

    if (!subscription) {
      return (
        <div style={{ textAlign: 'center', padding: 24 }}>
          <Empty
            description={t('billing.noSubscription', 'No active subscription')}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
          <Button
            type="primary"
            style={{ marginTop: 16 }}
            onClick={() => setUpgradeModalVisible(true)}
          >
            {t('billing.choosePlan', 'Choose a Plan')}
          </Button>
        </div>
      );
    }

    const plan = subscription.plan;
    return (
      <Space direction="vertical" style={{ width: '100%' }}>
        <div style={{ textAlign: 'center', padding: '16px 0' }}>
          <CrownOutlined style={{ fontSize: 48, color: '#faad14' }} />
          <Title level={3} style={{ margin: '16px 0 8px' }}>
            {plan?.display_name || subscription.plan_id}
          </Title>
          <Text type="secondary">{plan?.description}</Text>
        </div>

        <Statistic
          title={t('billing.monthlyPrice', 'Monthly Price')}
          value={plan?.price_monthly || 0}
          prefix="$"
          suffix="/mo"
        />

        <Divider />

        <div>
          <Text type="secondary">{t('billing.billingCycle', 'Billing Cycle')}: </Text>
          <Tag color="blue">{subscription.billing_cycle === 'yearly' ? t('billing.yearly', 'Yearly') : t('billing.monthly', 'Monthly')}</Tag>
        </div>

        <div>
          <Text type="secondary">{t('billing.currentPeriod', 'Current Period')}: </Text>
          <Text>{formatDate(subscription.current_period_start)} - {formatDate(subscription.current_period_end)}</Text>
        </div>

        {subscription.cancel_at_period_end && (
          <Alert
            type="warning"
            message={t('billing.cancelScheduled', 'Subscription will be canceled at the end of the billing period')}
            showIcon
          />
        )}

        <Space style={{ marginTop: 16 }}>
          <Button type="primary" onClick={() => setUpgradeModalVisible(true)}>
            {t('billing.changePlan', 'Change Plan')}
          </Button>
          {!subscription.cancel_at_period_end && (
            <Button danger onClick={() => setCancelModalVisible(true)}>
              {t('billing.cancelSubscription', 'Cancel')}
            </Button>
          )}
        </Space>
      </Space>
    );
  };

  // Render quota usage
  const renderQuotaUsage = () => {
    if (quotasLoading) {
      return <Spin />;
    }

    if (!quotas || quotas.length === 0) {
      return (
        <Empty
          description={t('billing.noQuotas', 'No quota information available')}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      );
    }

    return (
      <Row gutter={[16, 16]}>
        {quotas.map((quota: Quota) => {
          const percent = calculateUsagePercent(quota.used_value, quota.limit_value);
          const nearLimit = isQuotaNearLimit(quota.used_value, quota.limit_value);
          const exceeded = isQuotaExceeded(quota.used_value, quota.limit_value);
          const isUnlimited = quota.limit_value === -1;

          return (
            <Col xs={24} sm={12} key={quota.resource_type}>
              <Card size="small" hoverable>
                <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
                  <span style={{ fontSize: 20, marginRight: 8, color: '#1890ff' }}>
                    {resourceIcons[quota.resource_type]}
                  </span>
                  <Text strong>{getResourceTypeLabel(quota.resource_type)}</Text>
                  {nearLimit && !exceeded && (
                    <Tooltip title={t('billing.nearLimit', 'Approaching limit')}>
                      <ExclamationCircleOutlined style={{ marginLeft: 8, color: '#faad14' }} />
                    </Tooltip>
                  )}
                  {exceeded && (
                    <Tooltip title={t('billing.exceeded', 'Quota exceeded')}>
                      <ExclamationCircleOutlined style={{ marginLeft: 8, color: '#ff4d4f' }} />
                    </Tooltip>
                  )}
                </div>
                <div style={{ marginBottom: 8 }}>
                  <Text>
                    {quota.used_value.toFixed(1)} / {isUnlimited ? '∞' : quota.limit_value.toFixed(1)}
                  </Text>
                </div>
                <Progress
                  percent={isUnlimited ? 0 : percent}
                  status={exceeded ? 'exception' : nearLimit ? 'active' : 'normal'}
                  strokeColor={exceeded ? '#ff4d4f' : nearLimit ? '#faad14' : '#1890ff'}
                  showInfo={!isUnlimited}
                />
                {quota.reset_at && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('billing.resetsOn', 'Resets on')}: {formatDate(quota.reset_at)}
                  </Text>
                )}
              </Card>
            </Col>
          );
        })}
      </Row>
    );
  };

  // Render usage summary (费用报告)
  const renderUsageSummary = () => {
    if (usageLoading) {
      return <Spin />;
    }

    if (!usageSummary || usageSummary.length === 0) {
      return (
        <Empty
          description={t('billing.noUsage', 'No usage data available')}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      );
    }

    const totalCost = usageSummary.reduce((sum: number, item: UsageSummary) => sum + item.cost, 0);

    return (
      <div>
        <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title={t('billing.totalUsageCost', 'Total Usage Cost')}
                value={totalCost}
                precision={2}
                prefix="$"
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title={t('billing.billingPeriod', 'Billing Period')}
                value={usageSummary[0]?.period_start ? formatDate(usageSummary[0].period_start) : '-'}
                suffix={usageSummary[0]?.period_end ? ` - ${formatDate(usageSummary[0].period_end)}` : ''}
                valueStyle={{ fontSize: 14 }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title={t('billing.resourceTypes', 'Resource Types')}
                value={usageSummary.length}
                suffix={t('billing.types', 'types')}
              />
            </Card>
          </Col>
        </Row>

        <Table
          dataSource={usageSummary}
          rowKey="resource_type"
          pagination={false}
          columns={[
            {
              title: t('billing.resourceType', 'Resource Type'),
              dataIndex: 'resource_type',
              key: 'resource_type',
              render: (type: ResourceType) => (
                <Space>
                  {resourceIcons[type]}
                  <span>{getResourceTypeLabel(type)}</span>
                </Space>
              ),
            },
            {
              title: t('billing.usage', 'Usage'),
              key: 'usage',
              render: (_: unknown, record: UsageSummary) => (
                <span>{record.total_usage.toFixed(2)} {record.unit}</span>
              ),
            },
            {
              title: t('billing.cost', 'Cost'),
              dataIndex: 'cost',
              key: 'cost',
              render: (cost: number) => formatCurrency(cost),
            },
          ]}
          summary={() => (
            <Table.Summary.Row>
              <Table.Summary.Cell index={0}>
                <Text strong>{t('billing.total', 'Total')}</Text>
              </Table.Summary.Cell>
              <Table.Summary.Cell index={1} />
              <Table.Summary.Cell index={2}>
                <Text strong>{formatCurrency(totalCost)}</Text>
              </Table.Summary.Cell>
            </Table.Summary.Row>
          )}
        />
      </div>
    );
  };

  // Render plan comparison (套餐对比)
  const renderPlanComparison = () => {
    if (plansLoading) {
      return <Spin />;
    }

    if (!plans || plans.length === 0) {
      return (
        <Empty
          description={t('billing.noPlans', 'No plans available')}
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      );
    }

    return (
      <div>
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

        <Row gutter={24}>
          {plans.map((plan: Plan) => {
            const isCurrentPlan = subscription?.plan_id === plan.id;
            const price = billingCycle === 'yearly' 
              ? Math.round(plan.price_yearly / 12) 
              : plan.price_monthly;

            return (
              <Col xs={24} sm={12} lg={8} xl={6} key={plan.id}>
                <Card
                  hoverable
                  style={{
                    borderColor: isCurrentPlan ? '#1890ff' : selectedPlan === plan.id ? '#52c41a' : undefined,
                    borderWidth: isCurrentPlan || selectedPlan === plan.id ? 2 : 1,
                  }}
                  onClick={() => !isCurrentPlan && handleSelectPlan(plan.id)}
                >
                  {isCurrentPlan && (
                    <Tag color="blue" style={{ position: 'absolute', top: 12, right: 12 }}>
                      {t('billing.currentPlan', 'Current')}
                    </Tag>
                  )}
                  
                  <div style={{ textAlign: 'center', marginBottom: 16 }}>
                    <Title level={4}>{plan.display_name}</Title>
                    <Title level={2} style={{ margin: '16px 0', color: '#1890ff' }}>
                      ${price}
                      <Text type="secondary" style={{ fontSize: 14 }}>/mo</Text>
                    </Title>
                    <Text type="secondary">{plan.description}</Text>
                  </div>

                  <Divider />

                  <Space direction="vertical" style={{ width: '100%' }}>
                    <div>
                      <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                      {plan.limits.max_environments === -1 
                        ? t('billing.unlimited', 'Unlimited') 
                        : plan.limits.max_environments} {t('billing.environments', 'environments')}
                    </div>
                    <div>
                      <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                      {plan.limits.cpu_hours_monthly === -1 
                        ? t('billing.unlimited', 'Unlimited') 
                        : plan.limits.cpu_hours_monthly} {t('billing.cpuHours', 'CPU hours/mo')}
                    </div>
                    <div>
                      <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                      {plan.limits.storage_gb === -1 
                        ? t('billing.unlimited', 'Unlimited') 
                        : `${plan.limits.storage_gb}GB`} {t('billing.storage', 'storage')}
                    </div>
                    {plan.features.collaboration && (
                      <div>
                        <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                        {t('billing.collaboration', 'Real-time collaboration')}
                      </div>
                    )}
                    {plan.features.custom_domains && (
                      <div>
                        <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                        {t('billing.customDomains', 'Custom domains')}
                      </div>
                    )}
                    {plan.features.sso && (
                      <div>
                        <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 8 }} />
                        {t('billing.sso', 'SSO integration')}
                      </div>
                    )}
                    <div>
                      <InfoCircleOutlined style={{ color: '#1890ff', marginRight: 8 }} />
                      {t(`billing.support.${plan.features.support}`, plan.features.support)} {t('billing.support', 'support')}
                    </div>
                  </Space>
                </Card>
              </Col>
            );
          })}
        </Row>
      </div>
    );
  };

  // Main render
  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>{t('billing.title')}</Title>

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab={t('billing.overview', 'Overview')} key="overview">
          <Row gutter={[24, 24]}>
            {/* Current Plan */}
            <Col xs={24} lg={8}>
              <Card title={t('billing.currentPlan')} style={{ height: '100%' }}>
                {renderCurrentPlan()}
              </Card>
            </Col>

            {/* Quota Usage */}
            <Col xs={24} lg={16}>
              <Card 
                title={t('billing.quotaUsage', 'Quota Usage')}
                extra={
                  <Tooltip title={t('billing.quotaInfo', 'Your resource usage limits based on your plan')}>
                    <InfoCircleOutlined />
                  </Tooltip>
                }
              >
                {renderQuotaUsage()}
              </Card>
            </Col>
          </Row>
        </TabPane>

        <TabPane tab={t('billing.usageReport', 'Usage Report')} key="usage">
          <Card title={t('billing.usageReport', 'Usage Report')}>
            {renderUsageSummary()}
          </Card>
        </TabPane>

        <TabPane tab={t('billing.plans', 'Plans')} key="plans">
          <Card 
            title={t('billing.comparePlans', 'Compare Plans')}
            extra={
              <Button type="link" onClick={() => navigate('/billing/plans')}>
                {t('billing.viewDetailedComparison', 'View Detailed Comparison')}
              </Button>
            }
          >
            {renderPlanComparison()}
            
            {selectedPlan && selectedPlan !== subscription?.plan_id && (
              <div style={{ marginTop: 24, textAlign: 'center' }}>
                <Button
                  type="primary"
                  size="large"
                  onClick={handleConfirmPlanChange}
                  loading={changePlanMutation.isPending || createSubscriptionMutation.isPending}
                >
                  {subscription 
                    ? t('billing.confirmChange', 'Confirm Plan Change')
                    : t('billing.subscribe', 'Subscribe Now')}
                </Button>
              </div>
            )}
          </Card>
        </TabPane>

        <TabPane tab={t('billing.invoices')} key="invoices">
          <Card title={t('billing.invoices')}>
            <Table
              dataSource={invoicesData?.data || []}
              columns={invoiceColumns}
              rowKey="id"
              loading={invoicesLoading}
              pagination={{
                total: invoicesData?.pagination.total || 0,
                pageSize: invoicesData?.pagination.page_size || 20,
                current: invoicesData?.pagination.page || 1,
              }}
            />
          </Card>
        </TabPane>
      </Tabs>

      {/* Upgrade Modal */}
      <Modal
        title={t('billing.changePlan', 'Change Plan')}
        open={upgradeModalVisible}
        onCancel={() => {
          setUpgradeModalVisible(false);
          setSelectedPlan(null);
        }}
        footer={null}
        width={1000}
      >
        {renderPlanComparison()}
        
        {selectedPlan && selectedPlan !== subscription?.plan_id && (
          <div style={{ marginTop: 24, textAlign: 'center' }}>
            <Button
              type="primary"
              size="large"
              onClick={handleConfirmPlanChange}
              loading={changePlanMutation.isPending || createSubscriptionMutation.isPending}
            >
              {subscription 
                ? t('billing.confirmChange', 'Confirm Plan Change')
                : t('billing.subscribe', 'Subscribe Now')}
            </Button>
          </div>
        )}
      </Modal>

      {/* Cancel Subscription Modal */}
      <Modal
        title={t('billing.cancelSubscription', 'Cancel Subscription')}
        open={cancelModalVisible}
        onCancel={() => setCancelModalVisible(false)}
        footer={[
          <Button key="back" onClick={() => setCancelModalVisible(false)}>
            {t('common.cancel')}
          </Button>,
          <Button
            key="end"
            onClick={() => cancelSubscriptionMutation.mutate(false)}
            loading={cancelSubscriptionMutation.isPending}
          >
            {t('billing.cancelAtEnd', 'Cancel at Period End')}
          </Button>,
          <Button
            key="immediate"
            danger
            onClick={() => cancelSubscriptionMutation.mutate(true)}
            loading={cancelSubscriptionMutation.isPending}
          >
            {t('billing.cancelImmediately', 'Cancel Immediately')}
          </Button>,
        ]}
      >
        <Paragraph>
          {t('billing.cancelWarning', 'Are you sure you want to cancel your subscription?')}
        </Paragraph>
        <Alert
          type="warning"
          message={t('billing.cancelInfo', 'You will lose access to premium features when your subscription ends.')}
          showIcon
        />
      </Modal>

      {/* Payment Modal */}
      {selectedInvoice && (
        <PaymentModal
          visible={paymentModalVisible}
          onClose={() => {
            setPaymentModalVisible(false);
            setSelectedInvoice(null);
          }}
          onSuccess={handlePaymentSuccess}
          invoiceId={selectedInvoice.id}
          amount={selectedInvoice.amount_due}
          currency={selectedInvoice.currency}
          description={`${t('payment.paymentFor', 'Payment for')} ${selectedInvoice.invoice_number}`}
        />
      )}
    </div>
  );
};

export default Billing;
