/**
 * Timezone Selector Component
 */

import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Select, Space } from 'antd';
import { ClockCircleOutlined } from '@ant-design/icons';
import {
  timezones,
  getUserTimezone,
  setUserTimezone,
  getTimezoneOffset,
} from '../i18n/timezone';

interface TimezoneSelectorProps {
  showLabel?: boolean;
  size?: 'small' | 'middle' | 'large';
  className?: string;
  onChange?: (timezone: string) => void;
}

export const TimezoneSelector: React.FC<TimezoneSelectorProps> = ({
  showLabel = false,
  size = 'middle',
  className,
  onChange,
}) => {
  const { t } = useTranslation();
  const [timezone, setTimezone] = useState(getUserTimezone());

  useEffect(() => {
    setTimezone(getUserTimezone());
  }, []);

  const handleChange = (value: string) => {
    setTimezone(value);
    setUserTimezone(value);
    onChange?.(value);
  };

  const options = timezones.map(tz => ({
    value: tz.value,
    label: (
      <Space>
        <span>{tz.label}</span>
        <span style={{ color: '#999', fontSize: '12px' }}>
          ({getTimezoneOffset(tz.value)})
        </span>
      </Space>
    ),
  }));

  return (
    <Space className={className}>
      {showLabel && (
        <span>
          <ClockCircleOutlined /> {t('settings.preferences.timezone')}:
        </span>
      )}
      <Select
        value={timezone}
        onChange={handleChange}
        options={options}
        size={size}
        style={{ minWidth: 280 }}
        showSearch
        filterOption={(input, option) =>
          (option?.label as any)?.props?.children?.[0]?.props?.children
            ?.toLowerCase()
            .includes(input.toLowerCase()) ?? false
        }
        suffixIcon={<ClockCircleOutlined />}
      />
    </Space>
  );
};

export default TimezoneSelector;
