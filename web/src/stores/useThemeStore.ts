import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { Theme } from '@/types';
import { STORAGE_KEY_THEME } from '@/utils/constants';

interface ThemeState {
  theme: Theme;
  resolvedTheme: Theme;
  setTheme: (theme: Theme) => void;
  cycleTheme: () => void;
  initializeTheme: () => () => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: 'light',
      resolvedTheme: 'light',

      setTheme: () => {
        document.documentElement.classList.remove('dark');
        set({ theme: 'light', resolvedTheme: 'light' });
      },

      cycleTheme: () => {},

      initializeTheme: () => {
        document.documentElement.classList.remove('dark');
        return () => {};
      },
    }),
    {
      name: STORAGE_KEY_THEME,
    }
  )
);
