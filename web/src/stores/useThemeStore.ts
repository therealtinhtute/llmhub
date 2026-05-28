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

const applyTheme = (theme: Theme) => {
  if (theme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
};

export const useThemeStore = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: 'light',
      resolvedTheme: 'light',

      setTheme: (theme) => {
        applyTheme(theme);
        set({ theme, resolvedTheme: theme });
      },

      cycleTheme: () => {
        const { theme, setTheme } = get();
        setTheme(theme === 'light' ? 'dark' : 'light');
      },

      initializeTheme: () => {
        const { theme, setTheme } = get();
        setTheme(theme);
        return () => {};
      },
    }),
    {
      name: STORAGE_KEY_THEME,
    }
  )
);
