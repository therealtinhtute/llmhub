import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { Navigate, useNavigate, useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { AnimatePresence, motion, useReducedMotion } from 'motion/react';
import { Button } from '@/components/ui/Button';
import { FormInput as Input } from '@/components/ui/FormInput';
import { FormSelect as Select } from '@/components/ui/FormSelect';
import { SelectionCheckbox } from '@/components/ui/SelectionCheckbox';
import { IconEye, IconEyeOff } from '@/components/ui/icons';
import { toast } from 'sonner';
import { useAuthStore, useLanguageStore } from '@/stores';
import { normalizeApiBase, resolvePreferredApiBase } from '@/utils/connection';
import { LANGUAGE_LABEL_KEYS, LANGUAGE_ORDER } from '@/utils/constants';
import { isSupportedLanguage } from '@/utils/language';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';
import type { ApiError } from '@/types';

/**
 * 将 API 错误转换为本地化的用户友好消息
 */
type RedirectState = { from?: { pathname?: string } };

const BRAND_WORDS = [
  { word: 'CLI', finalOpacity: 0.95 },
  { word: 'PROXY', finalOpacity: 0.7 },
  { word: 'API', finalOpacity: 0.45 },
];

function getLocalizedErrorMessage(error: unknown, t: (key: string) => string): string {
  const apiError = error as Partial<ApiError>;
  const status = typeof apiError.status === 'number' ? apiError.status : undefined;
  const code = typeof apiError.code === 'string' ? apiError.code : undefined;
  const message =
    error instanceof Error
      ? error.message
      : typeof apiError.message === 'string'
        ? apiError.message
        : typeof error === 'string'
          ? error
          : '';

  const withHttpStatus = (summary: string) => {
    if (!status) {
      return summary;
    }

    const genericAxiosMessage = `Request failed with status code ${status}`;
    const detail = message.trim();
    const backendDetail =
      detail && detail !== genericAxiosMessage
        ? ` (${t('login.error_backend_detail')}: ${detail})`
        : '';

    return `HTTP ${status}: ${summary}${backendDetail}`;
  };

  // 根据 HTTP 状态码判断
  if (status === 401) {
    return withHttpStatus(t('login.error_unauthorized'));
  }
  if (status === 403) {
    return withHttpStatus(t('login.error_forbidden'));
  }
  if (status === 404) {
    return withHttpStatus(t('login.error_not_found'));
  }
  if (status && status >= 500) {
    return withHttpStatus(t('login.error_server'));
  }

  // 根据 axios 错误码判断
  if (code === 'ECONNABORTED' || message.toLowerCase().includes('timeout')) {
    return t('login.error_timeout');
  }
  if (code === 'ERR_NETWORK' || message.toLowerCase().includes('network error')) {
    return t('login.error_network');
  }
  if (code === 'ERR_CERT_AUTHORITY_INVALID' || message.toLowerCase().includes('certificate')) {
    return t('login.error_ssl');
  }

  // 检查 CORS 错误
  if (message.toLowerCase().includes('cors') || message.toLowerCase().includes('cross-origin')) {
    return t('login.error_cors');
  }

  // 默认错误消息
  return withHttpStatus(t('login.error_invalid'));
}

export function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const reduceMotion = useReducedMotion();
  const language = useLanguageStore((state) => state.language);
  const setLanguage = useLanguageStore((state) => state.setLanguage);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const login = useAuthStore((state) => state.login);
  const restoreSession = useAuthStore((state) => state.restoreSession);
  const storedBase = useAuthStore((state) => state.apiBase);
  const storedKey = useAuthStore((state) => state.managementKey);
  const storedRememberPassword = useAuthStore((state) => state.rememberPassword);

  const [apiBase, setApiBase] = useState('');
  const [managementKey, setManagementKey] = useState('');
  const [showCustomBase, setShowCustomBase] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const [rememberPassword, setRememberPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [autoLoading, setAutoLoading] = useState(true);
  const [autoLoginSuccess, setAutoLoginSuccess] = useState(false);
  const [error, setError] = useState('');

  const detectedBase = useMemo(() => resolvePreferredApiBase(), []);
  const languageOptions = useMemo(
    () =>
      LANGUAGE_ORDER.map((lang) => ({
        value: lang,
        label: t(LANGUAGE_LABEL_KEYS[lang])
      })),
    [t]
  );
  const handleLanguageChange = useCallback(
    (selectedLanguage: string) => {
      if (!isSupportedLanguage(selectedLanguage)) {
        return;
      }
      setLanguage(selectedLanguage);
    },
    [setLanguage]
  );

  useEffect(() => {
    const init = async () => {
      try {
        const autoLoggedIn = await restoreSession();
        if (autoLoggedIn) {
          setAutoLoginSuccess(true);
          // 延迟跳转，让用户看到成功动画
          setTimeout(() => {
            const redirect = (location.state as RedirectState | null)?.from?.pathname || '/';
            navigate(redirect, { replace: true });
          }, 1500);
        } else {
          setApiBase(storedBase || detectedBase);
          setManagementKey(storedKey || '');
          setRememberPassword(storedRememberPassword || Boolean(storedKey));
        }
      } finally {
        if (!autoLoginSuccess) {
          setAutoLoading(false);
        }
      }
    };

    init();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!managementKey.trim()) {
      setError(t('login.error_required'));
      return;
    }

    const baseToUse = apiBase ? normalizeApiBase(apiBase) : detectedBase;
    setLoading(true);
    setError('');
    try {
      await login({
        apiBase: baseToUse,
        managementKey: managementKey.trim(),
        rememberPassword
      });
      toast.success(t('common.connected_status'));
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const message = getLocalizedErrorMessage(err, t);
      setError(message);
      toast.error(`${t('notification.login_failed')}: ${message}`);
    } finally {
      setLoading(false);
    }
  }, [apiBase, detectedBase, login, managementKey, navigate, rememberPassword, t]);

  const handleSubmitKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === 'Enter' && !loading) {
        event.preventDefault();
        handleSubmit();
      }
    },
    [loading, handleSubmit]
  );

  if (isAuthenticated && !autoLoading && !autoLoginSuccess) {
    const redirect = (location.state as RedirectState | null)?.from?.pathname || '/';
    return <Navigate to={redirect} replace />;
  }

  // 显示启动动画（自动登录中或自动登录成功）
  const showSplash = autoLoading || autoLoginSuccess;

  return (
    <div className="min-h-screen flex">
      {/* 左侧品牌展示区 */}
      <div className="flex-1 flex flex-col justify-center items-center bg-black px-8 py-8 relative overflow-hidden max-md:hidden">
        <div className="relative z-[1] flex flex-col items-end w-full">
          {BRAND_WORDS.map(({ word, finalOpacity }, index) => (
            <motion.span
              key={word}
              className="text-[14vw] font-black text-white tracking-tight leading-[0.85] uppercase text-right"
              style={{ opacity: reduceMotion ? finalOpacity : 0 }}
              initial={reduceMotion ? false : { opacity: 0, x: 24 }}
              animate={{ opacity: finalOpacity, x: 0 }}
              transition={{ duration: 0.5, delay: 0.15 + index * 0.09, ease: [0.23, 1, 0.32, 1] }}
            >
              {word}
            </motion.span>
          ))}
        </div>
      </div>

      {/* 右侧功能交互区 */}
      <div className="flex-1 flex flex-col justify-center items-center p-8 bg-background relative max-md:p-6 max-md:min-h-screen">
        <AnimatePresence mode="wait" initial={false}>
          {showSplash ? (
            /* 启动动画 */
            <motion.div
              key="splash"
              className="flex flex-col items-center gap-4"
              exit={reduceMotion ? undefined : { opacity: 0, scale: 0.98 }}
              transition={{ duration: 0.2, ease: 'easeOut' }}
            >
              <img src={INLINE_LOGO_JPEG} alt="LLMHUB" className="h-30 w-auto" />
              <h1 className="text-[28px] font-serif italic font-semibold text-foreground m-0 tracking-tight">{t('splash.title')}</h1>
              <p className="text-base font-medium text-muted-foreground m-0 -mt-2">{t('splash.subtitle')}</p>
              <div className="w-[120px] h-[3px] bg-border overflow-hidden rounded-full mt-4">
                {autoLoginSuccess ? (
                  <motion.div
                    className="h-full bg-primary rounded-full"
                    initial={{ width: '0%' }}
                    animate={{ width: '100%' }}
                    transition={{ duration: 0.45, ease: [0.23, 1, 0.32, 1] }}
                  />
                ) : (
                  <motion.div
                    className="h-full w-1/3 bg-primary/70 rounded-full"
                    animate={reduceMotion ? undefined : { x: ['-40px', '120px'] }}
                    transition={
                      reduceMotion
                        ? undefined
                        : { duration: 1.1, repeat: Infinity, ease: 'easeInOut', repeatDelay: 0.15 }
                    }
                    style={reduceMotion ? { opacity: 0.6 } : undefined}
                  />
                )}
              </div>
            </motion.div>
          ) : (
            /* 登录表单 */
            <motion.div
              key="login-form"
              className="w-full max-w-[420px] flex flex-col items-center gap-6"
              initial={reduceMotion ? false : { opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.35, ease: [0.23, 1, 0.32, 1] }}
            >
            {/* Logo */}
            <img src={INLINE_LOGO_JPEG} alt="Logo" className="w-20 h-20 object-cover shadow-lg border-[3px] border-border" />

            {/* 登录表单卡片 */}
            <div className="w-full bg-card border border-border shadow-lg p-6 flex flex-col gap-4 max-md:p-4 max-md:shadow-none max-md:border-0 max-md:bg-transparent">
              <div className="flex flex-col gap-2 text-center">
                <div className="flex items-center justify-center gap-2 flex-wrap">
                  <div className="text-[22px] font-serif italic font-semibold text-foreground">{t('title.login')}</div>
                  <Select
                    className="min-w-[108px] shrink-0"
                    value={language}
                    options={languageOptions}
                    onChange={handleLanguageChange}
                    fullWidth={false}
                    ariaLabel={t('language.switch')}
                  />
                </div>
                <div className="text-muted-foreground text-sm">{t('login.subtitle')}</div>
              </div>

              <div className="bg-muted border border-dashed border-border p-3 flex flex-col gap-1">
                <div className="text-muted-foreground text-sm">{t('login.connection_current')}</div>
                <div className="font-bold text-foreground break-all">{apiBase || detectedBase}</div>
                <div className="text-muted-foreground text-xs">{t('login.connection_auto_hint')}</div>
              </div>

              <div className="flex justify-start w-full">
                <SelectionCheckbox
                  checked={showCustomBase}
                  onChange={setShowCustomBase}
                  ariaLabel={t('login.custom_connection_label')}
                  label={t('login.custom_connection_label')}
                  labelClassName="text-muted-foreground text-sm font-medium"
                />
              </div>

              {showCustomBase && (
                <Input
                  label={t('login.custom_connection_label')}
                  placeholder={t('login.custom_connection_placeholder')}
                  value={apiBase}
                  onChange={(e) => setApiBase(e.target.value)}
                  hint={t('login.custom_connection_hint')}
                />
              )}

              <Input
                autoFocus
                label={t('login.management_key_label')}
                placeholder={t('login.management_key_placeholder')}
                type={showKey ? 'text' : 'password'}
                value={managementKey}
                onChange={(e) => setManagementKey(e.target.value)}
                onKeyDown={handleSubmitKeyDown}
                rightElement={
                  <button
                    type="button"
                    className="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors duration-150 hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    onClick={() => setShowKey((prev) => !prev)}
                    aria-label={
                      showKey
                        ? t('login.hide_key', { defaultValue: '隐藏密钥' })
                        : t('login.show_key', { defaultValue: '显示密钥' })
                    }
                    title={
                      showKey
                        ? t('login.hide_key', { defaultValue: '隐藏密钥' })
                        : t('login.show_key', { defaultValue: '显示密钥' })
                    }
                  >
                    {showKey ? <IconEyeOff size={16} /> : <IconEye size={16} />}
                  </button>
                }
              />

              <div className="flex justify-start w-full">
                <SelectionCheckbox
                  checked={rememberPassword}
                  onChange={setRememberPassword}
                  ariaLabel={t('login.remember_password_label')}
                  label={t('login.remember_password_label')}
                  labelClassName="text-muted-foreground text-sm font-medium"
                />
              </div>

              <Button fullWidth onClick={handleSubmit} loading={loading}>
                {loading ? t('login.submitting') : t('login.submit_button')}
              </Button>

              <AnimatePresence initial={false}>
                {error && (
                  <motion.div
                    className="bg-destructive/10 border border-destructive/40 px-3 py-2 text-destructive text-sm"
                    initial={reduceMotion ? false : { opacity: 0, y: -4 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={reduceMotion ? undefined : { opacity: 0 }}
                    transition={{ duration: 0.18, ease: 'easeOut' }}
                    role="alert"
                  >
                    {error}
                  </motion.div>
                )}
              </AnimatePresence>
            </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
