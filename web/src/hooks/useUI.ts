/**
 * RAD Gateway Admin UI - UI Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Custom hooks for UI state and interactions.
 */

import { useCallback, useEffect, useState, useSyncExternalStore } from 'react';
import { useUIStore, useSidebarCollapsed, useTheme, useActiveModal, useNotifications, useGlobalLoading } from '../stores/uiStore';
import { showNotification } from '../stores/uiStore';
import { Notification } from '../types';

/**
 * Hook to manage theme (light/dark/system).
 */
export function useThemeManager(): {
  theme: 'light' | 'dark' | 'system';
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  isDark: boolean;
  toggle: () => void;
} {
  const theme = useTheme();
  const store = useUIStore();

  const isDark = theme === 'dark' || (theme === 'system' && getSystemTheme() === 'dark');

  const toggle = useCallback(() => {
    const nextTheme = isDark ? 'light' : 'dark';
    store.setTheme(nextTheme);
  }, [isDark, store]);

  const setTheme = useCallback(
    (newTheme: 'light' | 'dark' | 'system') => {
      store.setTheme(newTheme);
    },
    [store]
  );

  // Watch system theme changes
  useEffect(() => {
    if (theme !== 'system') return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => {
      // Re-apply theme
      store.setTheme('system');
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [theme, store]);

  return {
    theme,
    setTheme,
    isDark,
    toggle,
  };
}

function getSystemTheme(): 'light' | 'dark' {
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/**
 * Hook to manage sidebar state.
 */
export function useSidebar(): {
  isCollapsed: boolean;
  toggle: () => void;
  setCollapsed: (collapsed: boolean) => void;
} {
  const isCollapsed = useSidebarCollapsed();
  const store = useUIStore();

  const toggle = useCallback(() => {
    store.toggleSidebar();
  }, [store]);

  const setCollapsed = useCallback(
    (collapsed: boolean) => {
      store.setSidebarCollapsed(collapsed);
    },
    [store]
  );

  return {
    isCollapsed,
    toggle,
    setCollapsed,
  };
}

/**
 * Hook to manage modal state.
 */
export function useModal(modalId: string): {
  isOpen: boolean;
  open: () => void;
  close: () => void;
  toggle: () => void;
} {
  const activeModal = useActiveModal();
  const store = useUIStore();
  const isOpen = activeModal === modalId;

  const open = useCallback(() => {
    store.openModal(modalId);
  }, [modalId, store]);

  const close = useCallback(() => {
    store.closeModal();
  }, [store]);

  const toggle = useCallback(() => {
    if (isOpen) {
      store.closeModal();
    } else {
      store.openModal(modalId);
    }
  }, [isOpen, modalId, store]);

  return {
    isOpen,
    open,
    close,
    toggle,
  };
}

/**
 * Hook to manage notifications.
 */
export function useNotificationsManager(): {
  notifications: Notification[];
  unreadCount: number;
  add: (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  remove: (id: string) => void;
  clear: () => void;
  showSuccess: (title: string, message?: string) => void;
  showError: (title: string, message?: string) => void;
  showWarning: (title: string, message?: string) => void;
  showInfo: (title: string, message?: string) => void;
} {
  const notifications = useNotifications();
  const store = useUIStore();

  const unreadCount = notifications.filter((n) => !n.read).length;

  const add = useCallback(
    (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => {
      store.addNotification(notification);
    },
    [store]
  );

  const markAsRead = useCallback(
    (id: string) => {
      store.markNotificationAsRead(id);
    },
    [store]
  );

  const markAllAsRead = useCallback(() => {
    store.markAllNotificationsAsRead();
  }, [store]);

  const remove = useCallback(
    (id: string) => {
      store.removeNotification(id);
    },
    [store]
  );

  const clear = useCallback(() => {
    store.clearNotifications();
  }, [store]);

  return {
    notifications,
    unreadCount,
    add,
    markAsRead,
    markAllAsRead,
    remove,
    clear,
    showSuccess: showNotification.success,
    showError: showNotification.error,
    showWarning: showNotification.warning,
    showInfo: showNotification.info,
  };
}

/**
 * Hook to manage loading state.
 */
export function useLoading(): {
  isLoading: boolean;
  setLoading: (loading: boolean) => void;
  withLoading: <T>(fn: () => Promise<T>) => Promise<T>;
} {
  const isLoading = useGlobalLoading();
  const store = useUIStore();

  const setLoading = useCallback(
    (loading: boolean) => {
      store.setLoading(loading);
    },
    [store]
  );

  const withLoading = useCallback(
    async <T>(fn: () => Promise<T>): Promise<T> => {
      setLoading(true);
      try {
        return await fn();
      } finally {
        setLoading(false);
      }
    },
    [setLoading]
  );

  return {
    isLoading,
    setLoading,
    withLoading,
  };
}

/**
 * Hook to debounce a value.
 */
export function useDebounce<T>(value: T, delay: number = 500): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);

  return debouncedValue;
}

/**
 * Hook to persist state in localStorage.
 */
export function useLocalStorage<T>(key: string, initialValue: T): [T, (value: T | ((prev: T) => T)) => void] {
  const [storedValue, setStoredValue] = useState<T>(() => {
    if (typeof window === 'undefined') {
      return initialValue;
    }

    try {
      const item = window.localStorage.getItem(key);
      return item ? (JSON.parse(item) as T) : initialValue;
    } catch (error) {
      console.error(`Error reading localStorage key "${key}":`, error);
      return initialValue;
    }
  });

  const setValue = useCallback(
    (value: T | ((prev: T) => T)) => {
      try {
        const valueToStore = value instanceof Function ? value(storedValue) : value;
        setStoredValue(valueToStore);

        if (typeof window !== 'undefined') {
          window.localStorage.setItem(key, JSON.stringify(valueToStore));
        }
      } catch (error) {
        console.error(`Error setting localStorage key "${key}":`, error);
      }
    },
    [key, storedValue]
  );

  return [storedValue, setValue];
}

/**
 * Hook to check media query.
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (callback: () => void) => {
      const matchMedia = window.matchMedia(query);
      matchMedia.addEventListener('change', callback);
      return () => {
        matchMedia.removeEventListener('change', callback);
      };
    },
    [query]
  );

  const getSnapshot = useCallback(() => {
    return window.matchMedia(query).matches;
  }, [query]);

  const getServerSnapshot = useCallback(() => {
    return false;
  }, []);

  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

/**
 * Hook to check if mobile viewport.
 */
export function useIsMobile(): boolean {
  return useMediaQuery('(max-width: 768px)');
}

/**
 * Hook to check if tablet viewport.
 */
export function useIsTablet(): boolean {
  return useMediaQuery('(min-width: 769px) and (max-width: 1024px)');
}

/**
 * Hook to check if desktop viewport.
 */
export function useIsDesktop(): boolean {
  return useMediaQuery('(min-width: 1025px)');
}
