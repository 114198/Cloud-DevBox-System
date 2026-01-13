/**
 * Locale hooks for i18n and timezone
 */

import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  changeLanguage,
  getCurrentLanguage,
  type SupportedLanguage,
} from '../i18n';
import {
  getUserTimezone,
  setUserTimezone,
  formatDate,
  formatTime,
  formatDateTime,
  formatRelativeTime,
} from '../i18n/timezone';

/**
 * Hook for language management
 */
export function useLanguage() {
  const { i18n, t } = useTranslation();

  const setLanguage = useCallback(async (lang: SupportedLanguage) => {
    await changeLanguage(lang);
  }, []);

  return {
    language: i18n.language as SupportedLanguage,
    setLanguage,
    t,
  };
}

/**
 * Hook for timezone management
 */
export function useTimezone() {
  const [timezone, setTimezoneState] = useState(getUserTimezone());

  const setTimezone = useCallback((tz: string) => {
    setUserTimezone(tz);
    setTimezoneState(tz);
  }, []);

  return {
    timezone,
    setTimezone,
  };
}

/**
 * Hook for date/time formatting
 */
export function useDateFormat() {
  const { timezone } = useTimezone();

  const format = useCallback(
    (date: Date | string | number, options?: Intl.DateTimeFormatOptions) => {
      return formatDateTime(date, options, timezone);
    },
    [timezone]
  );

  const formatDateOnly = useCallback(
    (date: Date | string | number, options?: Intl.DateTimeFormatOptions) => {
      return formatDate(date, options, timezone);
    },
    [timezone]
  );

  const formatTimeOnly = useCallback(
    (date: Date | string | number, options?: Intl.DateTimeFormatOptions) => {
      return formatTime(date, options, timezone);
    },
    [timezone]
  );

  const formatRelative = useCallback(
    (date: Date | string | number) => {
      return formatRelativeTime(date);
    },
    []
  );

  return {
    format,
    formatDate: formatDateOnly,
    formatTime: formatTimeOnly,
    formatRelative,
    timezone,
  };
}

/**
 * Hook for locale-aware number formatting
 */
export function useNumberFormat() {
  const { i18n } = useTranslation();

  const formatNumber = useCallback(
    (value: number, options?: Intl.NumberFormatOptions) => {
      return new Intl.NumberFormat(i18n.language, options).format(value);
    },
    [i18n.language]
  );

  const formatCurrency = useCallback(
    (value: number, currency = 'USD') => {
      return new Intl.NumberFormat(i18n.language, {
        style: 'currency',
        currency,
      }).format(value);
    },
    [i18n.language]
  );

  const formatPercent = useCallback(
    (value: number, decimals = 0) => {
      return new Intl.NumberFormat(i18n.language, {
        style: 'percent',
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(value);
    },
    [i18n.language]
  );

  const formatBytes = useCallback(
    (bytes: number, decimals = 2) => {
      if (bytes === 0) return '0 Bytes';

      const k = 1024;
      const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));

      return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${sizes[i]}`;
    },
    []
  );

  return {
    formatNumber,
    formatCurrency,
    formatPercent,
    formatBytes,
  };
}

/**
 * Combined locale hook
 */
export function useLocale() {
  const language = useLanguage();
  const timezone = useTimezone();
  const dateFormat = useDateFormat();
  const numberFormat = useNumberFormat();

  return {
    ...language,
    ...timezone,
    ...dateFormat,
    ...numberFormat,
  };
}

export default useLocale;
