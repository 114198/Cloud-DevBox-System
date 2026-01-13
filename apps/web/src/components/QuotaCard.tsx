/**
 * QuotaCard Component
 * Displays resource quota usage with visual indicators and upgrade prompts
 */

import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Progress,
  Typography,
  Tooltip,
  Button,
  Space,
} from 'antd';
import {
  ExclamationCircleOutlined,
  ThunderboltOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  RiseOutlined,
  ClockCircleOutlined,
  UpCircleOutlined,
} from '@ant-design/icons';
import { useDateFormat } from '../hooks/useLocale';
import {
  getResourceTypeLabel,
  calculateUsagePercent,
  isQuotaNearLimit,
  isQuotaExceeded,
  type Quota,
  type ResourceType,
} from '../services/billing';

const { Text } = Typography;

// Resource type icons
const resourceIcons: Record<ResourceType, React.ReactNode> = {
  cpu_hours: <ThunderboltOutlined />,
  memory_gb_hours: <CloudServerOutlined />,
  storage_gb_days: <DatabaseOutlined />,
  network_gb: <RiseOutlined />,
  build_minutes: <ClockCircleOutlined />,
};

interface QuotaCardProps {
  quota: Quota;
  onUpgrade?: () => void;
  showUpgradeButton?: boolean;
}

export const QuotaCard: React.FC<QuotaCardProps> = ({
  quota,
  onUpgrade,
  showUpgradeButton = true,
}) => {
  const { t } = useTranslation();
  const { formatDate } = useDateFormat();

  const percent = calculateUsagePercent(quota.used_value, quota.limit_value);
  const nearLimit = isQuotaNearLimit(quota.used_value, quota.limit_value);
  const exceeded = isQuotaExceeded(quota.used_value, quota.limit_value);
  const isUnlimited = quota.limit_value === -1;

  const getProgressStatus = () => {
    if (exceeded) return 'exception';
    if (nearLimit) return 'active';
    return 'normal';
  };

  const getProgressColor = () => {
    if (exceeded) return '#ff4d4f';
    if (nearLimit) return '#faad14';
    return '#1890ff';
  };

  return (
    <Card
      size="small"
      hoverable
      style={{
        borderColor: exceeded ? '#ff4d4f' : nearLimit ? '#faad14' : undefined,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 12 }}>
        <span style={{ fontSize: 20, marginRight: 8, color: getProgressColor() }}>
          {resourceIcons[quota.resource_type]}
        </span>
        <Text strong style={{ flex: 1 }}>
          {getResourceTypeLabel(quota.resource_type)}
        </Text>
        {nearLimit && !exceeded && (
          <Tooltip title={t('billing.nearLimit', 'Approaching limit')}>
            <ExclamationCircleOutlined style={{ color: '#faad14' }} />
          </Tooltip>
        )}
        {exceeded && (
          <Tooltip title={t('billing.exceeded', 'Quota exceeded')}>
            <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
          </Tooltip>
        )}
      </div>

      <div style={{ marginBottom: 8 }}>
        <Text>
          {quota.used_value.toFixed(1)} / {isUnlimited ? '∞' : quota.limit_value.toFixed(1)}
        </Text>
        {!isUnlimited && (
          <Text type="secondary" style={{ float: 'right' }}>
            {percent.toFixed(0)}%
          </Text>
        )}
      </div>

      <Progress
        percent={isUnlimited ? 0 : percent}
        status={getProgressStatus()}
        strokeColor={getProgressColor()}
        showInfo={false}
      />

      <Space direction="vertical" style={{ width: '100%', marginTop: 8 }}>
        {quota.reset_at && (
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t('billing.resetsOn', 'Resets on')}: {formatDate(quota.reset_at)}
          </Text>
        )}

        {(nearLimit || exceeded) && showUpgradeButton && onUpgrade && (
          <Button
            type={exceeded ? 'primary' : 'default'}
            size="small"
            icon={<UpCircleOutlined />}
            onClick={onUpgrade}
            danger={exceeded}
            block
          >
            {exceeded
              ? t('billing.upgradeNow', 'Upgrade Now')
              : t('billing.considerUpgrade', 'Consider Upgrading')}
          </Button>
        )}
      </Space>
    </Card>
  );
};

export default QuotaCard;
