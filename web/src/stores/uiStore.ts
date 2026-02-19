/**
 * RAD Gateway Admin UI - UI Store
 * State Management Engineer - Phase 2 Implementation
 *
 * Zustand store for UI state (sidebar, theme, notifications, modals).
 * Handles global UI interactions and ephemeral state.
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { UIState, Notification } from '../types';

interface UIStore extends UIState {
  // Actions
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setTheme: (theme: UIState['theme']) => void;
  openModal: (modalId: string) => void;
  closeModal: () => void;
  toggleGlobalSearch: () => void;
  setGlobalSearchOpen: (open: boolean) => void;

  // Notification actions
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => void;
  markNotificationAsRead: (id: string) => void;
  markAllNotificationsAsRead: () => void;
  removeNotification: (id: string) => void;
  clearNotifications: () => void;
  getUnreadCount: () => number;

  // Loading actions
  setLoading: (loading: boolean) => void;
}

export const useUIStore = create<UIStore>()(
  persist(
    (set, get) => ({
      // Initial state
      sidebarCollapsed: false,
      theme: 'system',
      notifications: [],
      activeModal: null,
      globalSearchOpen: false,
      isLoading: false,

      // Toggle sidebar
      toggleSidebar: () => {
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }));
      },

      // Set sidebar collapsed state
      setSidebarCollapsed: (collapsed: boolean) => {
        set({ sidebarCollapsed: collapsed });
      },

      // Set theme
      setTheme: (theme: UIState['theme']) => {
        set({ theme });

        // Apply theme to document
        if (typeof document !== 'undefined') {
          const root = document.documentElement;

          if (theme === 'system') {
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
            root.classList.toggle('dark', prefersDark);
          } else {
            root.classList.toggle('dark', theme === 'dark');
          }
        }
      },

      // Open modal
      openModal: (modalId: string) => {
        set({ activeModal: modalId });

        // Prevent body scroll when modal is open
        if (typeof document !== 'undefined') {
          document.body.style.overflow = 'hidden';
        }
      },

      // Close modal
      closeModal: () => {
        set({ activeModal: null });

        // Restore body scroll
        if (typeof document !== 'undefined') {
          document.body.style.overflow = '';
        }
      },

      // Toggle global search
      toggleGlobalSearch: () => {
        set((state) => ({ globalSearchOpen: !state.globalSearchOpen }));
      },

      // Set global search open state
      setGlobalSearchOpen: (open: boolean) => {
        set({ globalSearchOpen: open });
      },

      // Add notification
      addNotification: (notification) => {
        const newNotification: Notification = {
          ...notification,
          id: `notif-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          timestamp: new Date().toISOString(),
          read: false,
        };

        set((state) => ({
          notifications: [newNotification, ...state.notifications].slice(0, 100), // Keep max 100
        }));
      },

      // Mark notification as read
      markNotificationAsRead: (id: string) => {
        set((state) => ({
          notifications: state.notifications.map((n) =>
            n.id === id ? { ...n, read: true } : n
          ),
        }));
      },

      // Mark all notifications as read
      markAllNotificationsAsRead: () => {
        set((state) => ({
          notifications: state.notifications.map((n) => ({ ...n, read: true })),
        }));
      },

      // Remove notification
      removeNotification: (id: string) => {
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        }));
      },

      // Clear all notifications
      clearNotifications: () => {
        set({ notifications: [] });
      },

      // Get unread count
      getUnreadCount: () => {
        return get().notifications.filter((n) => !n.read).length;
      },

      // Set loading
      setLoading: (loading: boolean) => {
        set({ isLoading: loading });
      },
    }),
    {
      name: 'rad-ui-storage',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        theme: state.theme,
      }),
    }
  )
);

// Selector hooks
export const useSidebarCollapsed = () => useUIStore((state) => state.sidebarCollapsed);
export const useTheme = () => useUIStore((state) => state.theme);
export const useNotifications = () => useUIStore((state) => state.notifications);
export const useUnreadNotificationCount = () =>
  useUIStore((state) => state.notifications.filter((n) => !n.read).length);
export const useActiveModal = () => useUIStore((state) => state.activeModal);
export const useGlobalSearchOpen = () => useUIStore((state) => state.globalSearchOpen);
export const useGlobalLoading = () => useUIStore((state) => state.isLoading);

// Helper to show common notification types
export const showNotification = {
  success: (title: string, message?: string, action?: Notification['action']) => {
    useUIStore.getState().addNotification({
      type: 'success',
      title,
      ...(message && { message }),
      ...(action && { action }),
    });
  },
  error: (title: string, message?: string, action?: Notification['action']) => {
    useUIStore.getState().addNotification({
      type: 'error',
      title,
      ...(message && { message }),
      ...(action && { action }),
    });
  },
  warning: (title: string, message?: string, action?: Notification['action']) => {
    useUIStore.getState().addNotification({
      type: 'warning',
      title,
      ...(message && { message }),
      ...(action && { action }),
    });
  },
  info: (title: string, message?: string, action?: Notification['action']) => {
    useUIStore.getState().addNotification({
      type: 'info',
      title,
      ...(message && { message }),
      ...(action && { action }),
    });
  },
};
