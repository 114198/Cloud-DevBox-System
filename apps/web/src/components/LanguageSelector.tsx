/**
 * Language Selector Component
 */

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Select, Space } from 'antd';
import { GlobalOutlined } from '@ant-design/icons';
import { supportedLanguages, changeLanguage, type SupportedLanguage } from '../i18n';

interface LanguageSelectorProps {
  showLabel?: boolean;
  size?: 'small' | 'middle' | 'large';
  className?: string;
}

export const LanguageSelector: React.FC<LanguageSelectorProps> = ({
  showLabel = false,
  size = 'middle',
  className,
}) => {
  const { i18n, t } = useTranslation();

  const handleChange = async (value: SupportedLanguage) => {
    await changeLanguage(value);
  };

  const options = supportedLanguages.map(lang => ({
    value: lang.code,
    label: (
      <Space>
        <span>{lang.nativeName}</span>
        {lang.code !== lang.nativeName && (
          <span style={{ color: '#999', fontSize: '12px' }}>({lang.name})</span>
        )}
      </Space>
    ),
  }));

  return (
    <Space className={className}>
      {showLabel && (
        <span>
          <GlobalOutlined /> {t('settings.preferences.language')}:
        </span>
      )}
      <Select
        value={i18n.language as SupportedLanguage}
        onChange={handleChange}
        options={options}
        size={size}
        style={{ minWidth: 150 }}
        suffixIcon={<GlobalOutlined />}
      />
    </Space>
  );
};

export default LanguageSelector;
