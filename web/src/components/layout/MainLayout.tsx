import {
  ReactNode,
  SVGProps,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { MainRoutes } from '@/router/MainRoutes';
import {
  IconSidebarApiKeys,
  IconSidebarAuthFiles,
  IconSidebarCombos,
  IconSidebarConfig,
  IconSidebarDashboard,
  IconSidebarLogs,
  IconSidebarModels,
  IconSidebarProviders,
  IconSidebarQuota,
  IconSidebarSystem,
} from '@/components/ui/icons';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';
import { toast } from 'sonner';
import {
  useAuthStore,
  useConfigStore,
  useLanguageStore,
  useThemeStore,
} from '@/stores';
import { triggerHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { LANGUAGE_LABEL_KEYS, LANGUAGE_ORDER } from '@/utils/constants';
import { isSupportedLanguage } from '@/utils/language';
import type { Theme } from '@/types';
import { Button } from '@/components/ui/Button';
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
} from '@/components/ui/sidebar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';

const sidebarIcons: Record<string, ReactNode> = {
  dashboard: <IconSidebarDashboard size={18} />,
  aiProviders: <IconSidebarProviders size={18} />,
  authFiles: <IconSidebarAuthFiles size={18} />,
  apiKeys: <IconSidebarApiKeys size={18} />,
  models: <IconSidebarModels size={18} />,
  combos: <IconSidebarCombos size={18} />,
  quota: <IconSidebarQuota size={18} />,
  config: <IconSidebarConfig size={18} />,
  logs: <IconSidebarLogs size={18} />,
  system: <IconSidebarSystem size={18} />,
};

const iconProps: SVGProps<SVGSVGElement> = {
  width: 16,
  height: 16,
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 2,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
  'aria-hidden': 'true',
  focusable: 'false',
};

const headerIcons = {
  refresh: (
    <svg {...iconProps}>
      <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
    </svg>
  ),
  language: (
    <svg {...iconProps}>
      <circle cx="12" cy="12" r="10" />
      <path d="M2 12h20" />
      <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
    </svg>
  ),
  sun: (
    <svg {...iconProps}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
    </svg>
  ),
  moon: (
    <svg {...iconProps}>
      <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z" />
    </svg>
  ),
  logout: (
    <svg {...iconProps}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <path d="m16 17 5-5-5-5" />
      <path d="M21 12H9" />
    </svg>
  ),
};

const THEME_CARDS: Array<{ key: Theme; labelKey: string }> = [
  { key: 'light', labelKey: 'theme.light' },
  { key: 'dark', labelKey: 'theme.dark' },
];

export function MainLayout() {
  const { t } = useTranslation();
  const location = useLocation();

  const logout = useAuthStore((state) => state.logout);
  const config = useConfigStore((state) => state.config);
  const fetchConfig = useConfigStore((state) => state.fetchConfig);
  const clearCache = useConfigStore((state) => state.clearCache);
  const theme = useThemeStore((state) => state.theme);
  const setTheme = useThemeStore((state) => state.setTheme);
  const language = useLanguageStore((state) => state.language);
  const setLanguage = useLanguageStore((state) => state.setLanguage);

  const contentRef = useRef<HTMLDivElement | null>(null);
  const headerRef = useRef<HTMLElement | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const isLogsPage = location.pathname.startsWith('/logs');

  // Set --header-height CSS var for mobile sticky elements and floating panels.
  useLayoutEffect(() => {
    const updateHeaderHeight = () => {
      const height = headerRef.current?.offsetHeight;
      if (height) {
        document.documentElement.style.setProperty('--header-height', `${height}px`);
      }
    };

    updateHeaderHeight();

    const resizeObserver =
      typeof ResizeObserver !== 'undefined' && headerRef.current
        ? new ResizeObserver(updateHeaderHeight)
        : null;
    if (resizeObserver && headerRef.current) {
      resizeObserver.observe(headerRef.current);
    }

    window.addEventListener('resize', updateHeaderHeight);

    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener('resize', updateHeaderHeight);
    };
  }, []);

  // Set --content-center-x CSS var for floating config panels and provider nav alignment.
  useLayoutEffect(() => {
    const updateContentCenter = () => {
      const el = contentRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      document.documentElement.style.setProperty(
        '--content-center-x',
        `${rect.left + rect.width / 2}px`
      );
    };

    updateContentCenter();

    const resizeObserver =
      typeof ResizeObserver !== 'undefined' && contentRef.current
        ? new ResizeObserver(updateContentCenter)
        : null;

    if (resizeObserver && contentRef.current) {
      resizeObserver.observe(contentRef.current);
    }

    window.addEventListener('resize', updateContentCenter);

    return () => {
      resizeObserver?.disconnect();
      window.removeEventListener('resize', updateContentCenter);
      document.documentElement.style.removeProperty('--content-center-x');
    };
  }, []);

  useEffect(() => {
    fetchConfig().catch(() => {});
  }, [fetchConfig]);

  const navGroups = [
    {
      id: 'operate',
      labelKey: 'nav_groups.operate',
      items: [
        { path: '/', labelKey: 'nav.dashboard', metaKey: 'nav_meta.dashboard', icon: sidebarIcons.dashboard },
      ],
    },
    {
      id: 'gateway',
      labelKey: 'nav_groups.gateway',
      items: [
        { path: '/ai-providers', labelKey: 'nav.ai_providers', metaKey: 'nav_meta.ai_providers', icon: sidebarIcons.aiProviders },
        { path: '/auth-files', labelKey: 'nav.auth_files', metaKey: 'nav_meta.auth_files', icon: sidebarIcons.authFiles },
        { path: '/api-keys', labelKey: 'nav.api_keys', metaKey: 'nav_meta.api_keys', icon: sidebarIcons.apiKeys },
        { path: '/models', labelKey: 'nav.models', metaKey: 'nav_meta.models', icon: sidebarIcons.models },
        { path: '/combos', labelKey: 'nav.combos', metaKey: 'nav_meta.combos', icon: sidebarIcons.combos },
      ],
    },
    {
      id: 'observe',
      labelKey: 'nav_groups.observe',
      items: [
        { path: '/quota', labelKey: 'nav.quota_management', metaKey: 'nav_meta.quota_management', icon: sidebarIcons.quota },
        { path: '/quota/monitoring', labelKey: 'nav.quota_monitoring', metaKey: 'nav_meta.quota_monitoring', icon: sidebarIcons.quota },
        ...(config?.loggingToFile
          ? [{ path: '/logs', labelKey: 'nav.logs', metaKey: 'nav_meta.logs', icon: sidebarIcons.logs }]
          : []),
      ],
    },
    {
      id: 'control',
      labelKey: 'nav_groups.control',
      items: [
        { path: '/config', labelKey: 'nav.config_management', metaKey: 'nav_meta.config_management', icon: sidebarIcons.config },
        { path: '/system', labelKey: 'nav.system_info', metaKey: 'nav_meta.system_info', icon: sidebarIcons.system },
      ],
    },
  ];

  const isItemActive = useCallback(
    (path: string) => {
      const normalized =
        location.pathname.length > 1 && location.pathname.endsWith('/')
          ? location.pathname.slice(0, -1)
          : location.pathname;
      const normalizedPath = normalized === '/dashboard' ? '/' : normalized;
      if (path === '/') return normalizedPath === '/';
      if (path === '/quota') return normalizedPath === '/quota';
      return normalizedPath === path || normalizedPath.startsWith(path + '/');
    },
    [location.pathname]
  );

  const handleRefreshAll = async () => {
    if (refreshing) return;
    setRefreshing(true);
    clearCache();
    try {
      const results = await Promise.allSettled([
        fetchConfig(undefined, true),
        triggerHeaderRefresh(),
      ]);
      const rejected = results.find((r) => r.status === 'rejected');
      if (rejected && rejected.status === 'rejected') {
        const reason = rejected.reason;
        const message =
          typeof reason === 'string' ? reason : reason instanceof Error ? reason.message : '';
        toast.error(`${t('notification.refresh_failed')}${message ? `: ${message}` : ''}`);
        return;
      }
      toast.success(t('notification.data_refreshed'));
    } finally {
      setRefreshing(false);
    }
  };

  const handleThemeSelect = useCallback(
    (nextTheme: string) => {
      if (nextTheme === 'light' || nextTheme === 'dark') {
        setTheme(nextTheme as Theme);
      }
    },
    [setTheme]
  );

  const handleLanguageSelect = useCallback(
    (nextLanguage: string) => {
      if (!isSupportedLanguage(nextLanguage)) return;
      setLanguage(nextLanguage);
    },
    [setLanguage]
  );

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader className="border-b border-sidebar-border p-3">
          <div className="grid justify-items-start gap-1" title="LLMHub Management Center">
            <img src={INLINE_LOGO_JPEG} alt="LLMHub logo" className="size-15 object-contain" />
            <span className="truncate font-serif text-xl font-semibold italic text-sidebar-foreground">
              {t('title.abbr')}
            </span>
          </div>
        </SidebarHeader>

        <SidebarContent>
          {navGroups.map((group) => (
            <SidebarGroup key={group.id}>
              <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {group.items.map((item) => (
                    <SidebarMenuItem key={item.path}>
                      <SidebarMenuButton
                        asChild
                        isActive={isItemActive(item.path)}
                        tooltip={t(item.labelKey)}
                      >
                        <NavLink to={item.path}>
                          {item.icon}
                          <span>{t(item.labelKey)}</span>
                        </NavLink>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          ))}
        </SidebarContent>

        <SidebarRail />
      </Sidebar>

      <SidebarInset>
        <header
          ref={headerRef}
          className="flex h-14 shrink-0 items-center gap-2 border-b border-border bg-background px-4"
        >
          <SidebarTrigger className="-ml-1 cursor-pointer" />
          <div className="ml-auto flex items-center gap-1">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleRefreshAll}
              disabled={refreshing}
              title={t('header.refresh_all')}
            >
              <span className={cn('inline-flex', refreshing && 'animate-spin')}>
                {headerIcons.refresh}
              </span>
            </Button>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  title={t('language.switch')}
                  aria-label={t('language.switch')}
                >
                  {headerIcons.language}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuRadioGroup value={language} onValueChange={handleLanguageSelect}>
                  {LANGUAGE_ORDER.map((lang) => (
                    <DropdownMenuRadioItem key={lang} value={lang}>
                      {t(LANGUAGE_LABEL_KEYS[lang])}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  title={t('theme.switch')}
                  aria-label={t('theme.switch')}
                >
                  {theme === 'dark' ? headerIcons.moon : headerIcons.sun}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuRadioGroup value={theme} onValueChange={handleThemeSelect}>
                  {THEME_CARDS.map((tc) => (
                    <DropdownMenuRadioItem key={tc.key} value={tc.key}>
                      {t(tc.labelKey)}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuContent>
            </DropdownMenu>

            <Button variant="ghost" size="sm" onClick={logout} title={t('header.logout')}>
              {headerIcons.logout}
            </Button>
          </div>
        </header>

        <div
          ref={contentRef}
          className={cn(
            'flex flex-1 flex-col min-w-0 overflow-y-auto [scrollbar-gutter:stable]',
            isLogsPage && 'overflow-hidden'
          )}
        >
          <div
            key={location.pathname}
            className={cn(
              'page-enter flex flex-1 flex-col gap-6 min-h-full px-[clamp(20px,3vw,48px)] pt-6 pb-10 overflow-x-hidden [&_h1]:font-serif [&_h1]:italic',
              isLogsPage && 'flex-auto min-h-0 overflow-hidden'
            )}
          >
            <MainRoutes />
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
