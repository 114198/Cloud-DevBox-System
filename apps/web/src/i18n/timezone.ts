/**
 * Timezone utilities for Cloud DevBox
 */

// Common timezones with display names
export const timezones = [
  { value: 'UTC', label: 'UTC (Coordinated Universal Time)', offset: 0 },
  { value: 'America/New_York', label: 'Eastern Time (US & Canada)', offset: -5 },
  { value: 'America/Chicago', label: 'Central Time (US & Canada)', offset: -6 },
  { value: 'America/Denver', label: 'Mountain Time (US & Canada)', offset: -7 },
  { value: 'America/Los_Angeles', label: 'Pacific Time (US & Canada)', offset: -8 },
  { value: 'America/Sao_Paulo', label: 'Brasilia Time', offset: -3 },
  { value: 'Europe/London', label: 'London (GMT)', offset: 0 },
  { value: 'Europe/Paris', label: 'Paris, Berlin, Rome', offset: 1 },
  { value: 'Europe/Moscow', label: 'Moscow', offset: 3 },
  { value: 'Asia/Dubai', label: 'Dubai', offset: 4 },
  { value: 'Asia/Kolkata', label: 'India Standard Time', offset: 5.5 },
  { value: 'Asia/Bangkok', label: 'Bangkok, Hanoi, Jakarta', offset: 7 },
  { value: 'Asia/Shanghai', label: 'Beijing, Shanghai, Hong Kong', offset: 8 },
  { value: 'Asia/Tokyo', label: 'Tokyo, Seoul', offset: 9 },
  { value: 'Australia/Sydney', label: 'Sydney, Melbourne', offset: 10 },
  { value: 'Pacific/Auckland', label: 'Auckland, Wellington', offset: 12 },
] as const;

export type TimezoneValue = typeof timezones[number]['value'];

// Storage key for user timezone preference
const TIMEZONE_STORAGE_KEY = 'devbox_timezone';

/**
 * Detect user's timezone from browser
 */
export function detectTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  } catch {
    return 'UTC';
  }
}

/**
 * Get user's preferred timezone (from storage or browser)
 */
export function getUserTimezone(): string {
  const stored = localStorage.getItem(TIMEZONE_STORAGE_KEY);
  if (stored && timezones.some(tz => tz.value === stored)) {
    return stored;
  }
  return detectTimezone();
}

/**
 * Set user's preferred timezone
 */
export function setUserTimezone(timezone: string): void {
  localStorage.setItem(TIMEZONE_STORAGE_KEY, timezone);
}

/**
 * Format date in user's timezone
 */
export function formatDate(
  date: Date | string | number,
  options?: Intl.DateTimeFormatOptions,
  timezone?: string
): string {
  const tz = timezone || getUserTimezone();
  const d = new Date(date);
  
  const defaultOptions: Intl.DateTimeFormatOptions = {
    timeZone: tz,
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    ...options,
  };

  return new Intl.DateTimeFormat(undefined, defaultOptions).format(d);
}

/**
 * Format time in user's timezone
 */
export function formatTime(
  date: Date | string | number,
  options?: Intl.DateTimeFormatOptions,
  timezone?: string
): string {
  const tz = timezone || getUserTimezone();
  const d = new Date(date);
  
  const defaultOptions: Intl.DateTimeFormatOptions = {
    timeZone: tz,
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  };

  return new Intl.DateTimeFormat(undefined, defaultOptions).format(d);
}

/**
 * Format date and time in user's timezone
 */
export function formatDateTime(
  date: Date | string | number,
  options?: Intl.DateTimeFormatOptions,
  timezone?: string
): string {
  const tz = timezone || getUserTimezone();
  const d = new Date(date);
  
  const defaultOptions: Intl.DateTimeFormatOptions = {
    timeZone: tz,
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  };

  return new Intl.DateTimeFormat(undefined, defaultOptions).format(d);
}

/**
 * Format relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(date: Date | string | number): string {
  const d = new Date(date);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);
  const diffWeek = Math.floor(diffDay / 7);

  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

  if (diffSec < 60) {
    return rtf.format(-diffSec, 'second');
  } else if (diffMin < 60) {
    return rtf.format(-diffMin, 'minute');
  } else if (diffHour < 24) {
    return rtf.format(-diffHour, 'hour');
  } else if (diffDay < 7) {
    return rtf.format(-diffDay, 'day');
  } else if (diffWeek < 4) {
    return rtf.format(-diffWeek, 'week');
  } else {
    return formatDate(d);
  }
}

/**
 * Get timezone offset string (e.g., "+08:00")
 */
export function getTimezoneOffset(timezone?: string): string {
  const tz = timezone || getUserTimezone();
  const date = new Date();
  const formatter = new Intl.DateTimeFormat('en-US', {
    timeZone: tz,
    timeZoneName: 'shortOffset',
  });
  
  const parts = formatter.formatToParts(date);
  const offsetPart = parts.find(p => p.type === 'timeZoneName');
  return offsetPart?.value || 'UTC';
}
