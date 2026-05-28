import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
} from 'react';
import { useTranslation } from 'react-i18next';
import { usePageTransitionLayer } from '@/components/common/PageTransitionLayer';
import { FormInput as Input } from '@/components/ui/FormInput';
import { Input as BaseInput } from '@/components/ui/Input';
import { FormSelect as Select } from '@/components/ui/FormSelect';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import {
  IconCode,
  IconDiamond,
  IconKey,
  IconSatellite,
  IconSettings,
  IconTimer,
  type IconProps,
} from '@/components/ui/icons';
import { ConfigSection } from '@/components/config/ConfigSection';
import { useMediaQuery } from '@/hooks/useMediaQuery';
import type {
  PayloadFilterRule,
  PayloadParamValidationErrorCode,
  PayloadRule,
  VisualConfigFieldPath,
  VisualConfigValidationErrorCode,
  VisualConfigValidationErrors,
  VisualConfigValues,
} from '@/types/visualConfig';
import {
  ApiKeysCardEditor,
  PayloadFilterRulesEditor,
  PayloadRulesEditor,
} from './VisualConfigEditorBlocks';

type VisualSectionId = 'server' | 'auth' | 'system' | 'quota' | 'streaming' | 'payload';

type VisualSection = {
  id: VisualSectionId;
  title: string;
  icon: ComponentType<IconProps>;
  errorCount: number;
};

interface VisualConfigEditorProps {
  values: VisualConfigValues;
  validationErrors?: VisualConfigValidationErrors;
  hasPayloadValidationErrors?: boolean;
  disabled?: boolean;
  onChange: (values: Partial<VisualConfigValues>) => void;
}

function getValidationMessage(
  t: ReturnType<typeof useTranslation>['t'],
  errorCode?: VisualConfigValidationErrorCode | PayloadParamValidationErrorCode
) {
  if (!errorCode) return undefined;
  return t(`config_management.visual.validation.${errorCode}`);
}

type ToggleRowProps = {
  title: string;
  description?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (value: boolean) => void;
};

function ToggleRow({ title, description, checked, disabled, onChange }: ToggleRowProps) {
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_auto] max-md:grid-cols-[minmax(0,1fr)] gap-[14px] items-center min-h-[74px] p-[14px] border border-border bg-transparent">
      <div className="flex flex-col gap-[5px] min-w-0">
        <div className="text-foreground text-[14px] font-bold leading-tight">{title}</div>
        {description ? (
          <div className="text-muted-foreground text-[12px] leading-[1.55]">{description}</div>
        ) : null}
      </div>
      <ToggleSwitch checked={checked} onChange={onChange} disabled={disabled} ariaLabel={title} />
    </div>
  );
}

function SectionGrid({ children }: { children: ReactNode }) {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] max-md:grid-cols-[minmax(0,1fr)] gap-[14px]">
      {children}
    </div>
  );
}

function SectionStack({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-[14px]">{children}</div>;
}

function Divider() {
  return <div className="h-px bg-border" />;
}

function SectionSubsection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 p-4 border border-border bg-transparent">
      <div className="flex flex-col gap-[5px]">
        <h3 className="m-0 text-foreground text-[15px] font-bold leading-tight">{title}</h3>
        {description ? (
          <p className="m-0 text-muted-foreground text-[12px] leading-[1.6]">{description}</p>
        ) : null}
      </div>
      {children}
    </div>
  );
}

function FieldShell({
  label,
  labelId,
  htmlFor,
  hint,
  hintId,
  error,
  errorId,
  children,
}: {
  label: string;
  labelId?: string;
  htmlFor?: string;
  hint?: string;
  hintId?: string;
  error?: string;
  errorId?: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-[7px] min-w-0">
      <label id={labelId} htmlFor={htmlFor} className="text-muted-foreground text-[12px] font-bold tracking-[0.02em]">
        {label}
      </label>
      {children}
      {error ? (
        <div id={errorId} className="p-[10px_14px] mb-2 bg-destructive/10 border border-destructive/35 text-destructive text-sm leading-[1.5]">
          {error}
        </div>
      ) : null}
      {hint ? (
        <div id={hintId} className="text-muted-foreground text-[12px] leading-[1.55]">
          {hint}
        </div>
      ) : null}
    </div>
  );
}

export function VisualConfigEditor({
  values,
  validationErrors,
  hasPayloadValidationErrors = false,
  disabled = false,
  onChange,
}: VisualConfigEditorProps) {
  const { t } = useTranslation();
  const pageTransitionLayer = usePageTransitionLayer();
  const isCurrentLayer = pageTransitionLayer ? pageTransitionLayer.isCurrentLayer : true;
  const isMobile = useMediaQuery('(max-width: 768px)');
  const routingStrategyLabelId = useId();
  const routingStrategyHintId = `${routingStrategyLabelId}-hint`;
  const disableImageGenerationLabelId = useId();
  const disableImageGenerationHintId = `${disableImageGenerationLabelId}-hint`;
  const keepaliveInputId = useId();
  const keepaliveHintId = `${keepaliveInputId}-hint`;
  const keepaliveErrorId = `${keepaliveInputId}-error`;
  const nonstreamKeepaliveInputId = useId();
  const nonstreamKeepaliveHintId = `${nonstreamKeepaliveInputId}-hint`;
  const nonstreamKeepaliveErrorId = `${nonstreamKeepaliveInputId}-error`;
  const [activeSectionId, setActiveSectionId] = useState<VisualSectionId>('server');
  const sectionRefs = useRef<Partial<Record<VisualSectionId, HTMLElement | null>>>({});
  const mobileNavScrollerRef = useRef<HTMLDivElement | null>(null);
  const mobileNavButtonRefs = useRef<Partial<Record<VisualSectionId, HTMLButtonElement | null>>>(
    {}
  );

  const isKeepaliveDisabled =
    values.streaming.keepaliveSeconds === '' || values.streaming.keepaliveSeconds === '0';
  const isNonstreamKeepaliveDisabled =
    values.streaming.nonstreamKeepaliveInterval === '' ||
    values.streaming.nonstreamKeepaliveInterval === '0';

  const portError = getValidationMessage(t, validationErrors?.port);
  const logsMaxSizeError = getValidationMessage(t, validationErrors?.logsMaxTotalSizeMb);
  const errorLogsMaxFilesError = getValidationMessage(t, validationErrors?.errorLogsMaxFiles);
  const redisUsageQueueRetentionError = getValidationMessage(
    t,
    validationErrors?.redisUsageQueueRetentionSeconds
  );
  const requestRetryError = getValidationMessage(t, validationErrors?.requestRetry);
  const maxRetryCredentialsError = getValidationMessage(t, validationErrors?.maxRetryCredentials);
  const maxRetryIntervalError = getValidationMessage(t, validationErrors?.maxRetryInterval);
  const authAutoRefreshWorkersError = getValidationMessage(
    t,
    validationErrors?.authAutoRefreshWorkers
  );
  const keepaliveError = getValidationMessage(t, validationErrors?.['streaming.keepaliveSeconds']);
  const bootstrapRetriesError = getValidationMessage(
    t,
    validationErrors?.['streaming.bootstrapRetries']
  );
  const nonstreamKeepaliveError = getValidationMessage(
    t,
    validationErrors?.['streaming.nonstreamKeepaliveInterval']
  );

  const handleApiKeysTextChange = useCallback(
    (apiKeysText: string) => onChange({ apiKeysText }),
    [onChange]
  );
  const handlePayloadDefaultRulesChange = useCallback(
    (payloadDefaultRules: PayloadRule[]) => onChange({ payloadDefaultRules }),
    [onChange]
  );
  const handlePayloadDefaultRawRulesChange = useCallback(
    (payloadDefaultRawRules: PayloadRule[]) => onChange({ payloadDefaultRawRules }),
    [onChange]
  );
  const handlePayloadOverrideRulesChange = useCallback(
    (payloadOverrideRules: PayloadRule[]) => onChange({ payloadOverrideRules }),
    [onChange]
  );
  const handlePayloadOverrideRawRulesChange = useCallback(
    (payloadOverrideRawRules: PayloadRule[]) => onChange({ payloadOverrideRawRules }),
    [onChange]
  );
  const handlePayloadFilterRulesChange = useCallback(
    (payloadFilterRules: PayloadFilterRule[]) => onChange({ payloadFilterRules }),
    [onChange]
  );
  const disableImageGenerationOptions = useMemo(
    () => [
      {
        value: 'false',
        label: t('config_management.visual.sections.network.disable_image_generation_false'),
      },
      {
        value: 'true',
        label: t('config_management.visual.sections.network.disable_image_generation_true'),
      },
      {
        value: 'chat',
        label: t('config_management.visual.sections.network.disable_image_generation_chat'),
      },
    ],
    [t]
  );

  const countErrors = useCallback(
    (fields: VisualConfigFieldPath[]) =>
      fields.reduce((total, field) => total + (validationErrors?.[field] ? 1 : 0), 0),
    [validationErrors]
  );

  const sections = useMemo<VisualSection[]>(
    () => [
      {
        id: 'server',
        title: t('config_management.visual.sections.server.title'),
        icon: IconSettings,
        errorCount: countErrors(['port']),
      },
      {
        id: 'auth',
        title: t('config_management.visual.sections.auth.title'),
        icon: IconKey,
        errorCount: 0,
      },
      {
        id: 'system',
        title: t('config_management.visual.sections.system.title'),
        icon: IconDiamond,
        errorCount: countErrors([
          'errorLogsMaxFiles',
          'logsMaxTotalSizeMb',
          'redisUsageQueueRetentionSeconds',
          'requestRetry',
          'maxRetryCredentials',
          'maxRetryInterval',
          'authAutoRefreshWorkers',
        ]),
      },
      {
        id: 'quota',
        title: t('config_management.visual.sections.quota.title'),
        icon: IconTimer,
        errorCount: 0,
      },
      {
        id: 'streaming',
        title: t('config_management.visual.sections.streaming.title'),
        icon: IconSatellite,
        errorCount: countErrors([
          'streaming.keepaliveSeconds',
          'streaming.bootstrapRetries',
          'streaming.nonstreamKeepaliveInterval',
        ]),
      },
      {
        id: 'payload',
        title: t('config_management.visual.sections.payload.title'),
        icon: IconCode,
        errorCount: hasPayloadValidationErrors ? 1 : 0,
      },
    ],
    [countErrors, hasPayloadValidationErrors, t]
  );

  const hasValidationIssues =
    sections.some((section) => section.errorCount > 0) || hasPayloadValidationErrors;
  const activeSection = sections.find((section) => section.id === activeSectionId) ?? sections[0];

  useEffect(() => {
    if (!isCurrentLayer) return undefined;
    if (typeof IntersectionObserver === 'undefined') return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        const visibleEntries = entries
          .filter((entry) => entry.isIntersecting)
          .sort((left, right) => right.intersectionRatio - left.intersectionRatio);

        if (visibleEntries.length === 0) return;
        setActiveSectionId(visibleEntries[0].target.id as VisualSectionId);
      },
      {
        rootMargin: '-18% 0px -58% 0px',
        threshold: [0.12, 0.3, 0.55],
      }
    );

    for (const section of sections) {
      const element = sectionRefs.current[section.id];
      if (element) observer.observe(element);
    }

    return () => observer.disconnect();
  }, [isCurrentLayer, sections]);

  useEffect(() => {
    if (!isCurrentLayer || !isMobile) return;
    const scroller = mobileNavScrollerRef.current;
    const button = mobileNavButtonRefs.current[activeSectionId];
    if (!scroller || !button) return;

    const scrollerRect = scroller.getBoundingClientRect();
    const buttonRect = button.getBoundingClientRect();
    const centeredLeft =
      scroller.scrollLeft +
      (buttonRect.left - scrollerRect.left) -
      (scroller.clientWidth - buttonRect.width) / 2;
    const maxScrollLeft = Math.max(scroller.scrollWidth - scroller.clientWidth, 0);
    const targetLeft = Math.min(Math.max(centeredLeft, 0), maxScrollLeft);

    scroller.scrollTo({
      left: targetLeft,
      behavior: 'smooth',
    });
  }, [activeSectionId, isCurrentLayer, isMobile]);

  const handleSectionJump = useCallback((sectionId: VisualSectionId) => {
    setActiveSectionId(sectionId);
    sectionRefs.current[sectionId]?.scrollIntoView({
      behavior: 'smooth',
      block: 'nearest',
      inline: 'start',
    });
  }, []);

  const navContent = (
    <div className="grid grid-cols-[repeat(3,minmax(0,1fr))] gap-2 min-w-0 max-[1200px]:grid-cols-[repeat(2,minmax(0,1fr))]">
      {sections.map((section, index) => {
        const Icon = section.icon;

        return (
          <button
            key={section.id}
            type="button"
            className={`flex items-center gap-[10px] w-full min-h-[48px] px-[11px] py-[9px] border text-left bg-transparent appearance-none cursor-pointer ${
              activeSectionId === section.id
                ? 'border-foreground bg-[color-mix(in_srgb,hsl(var(--foreground))_6%,transparent)]'
                : 'border-border hover:bg-[color-mix(in_srgb,hsl(var(--foreground))_5%,transparent)]'
            }`}
            onClick={() => handleSectionJump(section.id)}
          >
            <span className="min-w-[24px] pt-[2px] text-muted-foreground/60 text-[11px] font-[750] tracking-[0.08em] flex-none">
              {String(index + 1).padStart(2, '0')}
            </span>
            <span className="flex flex-col min-w-0 flex-1">
              <span className="flex items-center gap-2 min-w-0">
                <span className="inline-flex items-center gap-[7px] min-w-0">
                  <span className="inline-flex items-center justify-center w-4 text-muted-foreground flex-none">
                    <Icon size={14} />
                  </span>
                  <span className="text-foreground text-[13px] font-bold leading-tight">
                    {section.title}
                  </span>
                </span>
                {section.errorCount > 0 ? (
                  <span
                    className="inline-flex items-center justify-center min-w-[22px] h-[22px] px-[7px] bg-amber-100 border border-amber-400/30 text-amber-700 text-[11px] font-bold flex-none"
                    aria-hidden="true"
                  >
                    {section.errorCount}
                  </span>
                ) : null}
              </span>
            </span>
          </button>
        );
      })}
    </div>
  );

  return (
    <div className="flex flex-col gap-[18px]">
      <div className="grid grid-cols-[minmax(0,1fr)] gap-[10px] pb-[18px] max-md:pb-[14px] border-b border-border">
        <div className="flex items-center min-w-0 max-md:items-stretch">
          <div className="flex flex-wrap gap-1.5">
            <span className="inline-flex items-center min-h-[28px] px-[9px] border border-border text-muted-foreground text-[12px] font-bold leading-tight">
              {t('config_management.visual.quick_jump', { defaultValue: '快速跳转' })}
            </span>
            <span className="inline-flex items-center min-h-[28px] px-[9px] border border-border text-muted-foreground text-[12px] font-bold leading-tight">
              {activeSection?.title}
            </span>
            {hasValidationIssues ? (
              <span className="inline-flex items-center min-h-[28px] px-[9px] border border-amber-400/30 bg-amber-100 text-amber-700 text-[12px] font-bold leading-tight">
                {t('config_management.visual.validation.validation_blocked')}
              </span>
            ) : null}
          </div>
        </div>
      </div>

      <div className="flex flex-col gap-[14px] max-md:gap-3 min-w-0">
        {isMobile ? (
          <div className="sticky top-[calc(var(--header-height,64px)+10px)] z-[4] mb-1 bg-[color-mix(in_srgb,hsl(var(--muted))_92%,transparent)]">
            <div
              ref={mobileNavScrollerRef}
              className="grid grid-cols-[repeat(2,minmax(0,1fr))] gap-1.5 overflow-visible pt-[2px] pb-2"
              aria-label={t('config_management.visual.quick_jump', { defaultValue: '快速跳转' })}
            >
              {sections.map((section, index) => (
                <button
                  key={section.id}
                  ref={(node) => {
                    mobileNavButtonRefs.current[section.id] = node;
                  }}
                  type="button"
                  className={`inline-flex items-center gap-[7px] min-w-0 w-full px-[10px] py-[9px] border text-left bg-background appearance-none cursor-pointer ${
                    activeSectionId === section.id
                      ? 'border-foreground bg-foreground/[0.06]'
                      : 'border-border'
                  }`}
                  onClick={() => handleSectionJump(section.id)}
                >
                  <span className="text-muted-foreground/60 text-[11px] font-[750] tracking-[0.08em]">
                    {String(index + 1).padStart(2, '0')}
                  </span>
                  <span className="text-foreground text-[13px] font-bold leading-tight">
                    {section.title}
                  </span>
                  {section.errorCount > 0 ? (
                    <span
                      className="inline-flex items-center justify-center min-w-[20px] h-[20px] px-1.5 bg-amber-100 border border-amber-400/30 text-amber-700 text-[11px] font-bold"
                      aria-hidden="true"
                    >
                      {section.errorCount}
                    </span>
                  ) : null}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        <aside className="max-md:hidden sticky top-[calc(var(--header-height,64px)+12px)] z-[5] self-stretch min-w-0">
          <div className="pb-3 border-b border-border bg-[color-mix(in_srgb,hsl(var(--muted))_88%,transparent)]">
            {navContent}
          </div>
        </aside>

        <div className="flex gap-0 w-full max-w-full min-w-0 overflow-x-auto overflow-y-hidden items-stretch pb-3 max-md:pb-[10px] [scroll-padding-left:0] [scroll-snap-type:x_mandatory] [scrollbar-gutter:stable] [scrollbar-width:thin] [&>*]:flex-none [&>*]:w-full [&>*]:max-w-full">
          <ConfigSection
            id="server"
            ref={(node) => {
              sectionRefs.current.server = node;
            }}
            indexLabel="01"
            icon={<IconSettings size={16} />}
            title={t('config_management.visual.sections.server.title')}
            description={t('config_management.visual.sections.server.description')}
          >
            <SectionStack>
              <SectionGrid>
                <Input
                  label={t('config_management.visual.sections.server.host')}
                  placeholder="0.0.0.0"
                  value={values.host}
                  onChange={(e) => onChange({ host: e.target.value })}
                  disabled={disabled}
                />
                <Input
                  label={t('config_management.visual.sections.server.port')}
                  type="number"
                  placeholder="8317"
                  value={values.port}
                  onChange={(e) => onChange({ port: e.target.value })}
                  disabled={disabled}
                  error={portError}
                />
              </SectionGrid>

              <SectionSubsection
                title={t('config_management.visual.sections.tls.title')}
                description={t('config_management.visual.sections.tls.description')}
              >
                <SectionStack>
                  <ToggleRow
                    title={t('config_management.visual.sections.tls.enable')}
                    description={t('config_management.visual.sections.tls.enable_desc')}
                    checked={values.tlsEnable}
                    disabled={disabled}
                    onChange={(tlsEnable) => onChange({ tlsEnable })}
                  />

                  {values.tlsEnable ? (
                    <>
                      <Divider />
                      <SectionGrid>
                        <Input
                          label={t('config_management.visual.sections.tls.cert')}
                          placeholder="/path/to/cert.pem"
                          value={values.tlsCert}
                          onChange={(e) => onChange({ tlsCert: e.target.value })}
                          disabled={disabled}
                        />
                        <Input
                          label={t('config_management.visual.sections.tls.key')}
                          placeholder="/path/to/key.pem"
                          value={values.tlsKey}
                          onChange={(e) => onChange({ tlsKey: e.target.value })}
                          disabled={disabled}
                        />
                      </SectionGrid>
                    </>
                  ) : null}
                </SectionStack>
              </SectionSubsection>

              <SectionSubsection
                title={t('config_management.visual.sections.remote.title')}
                description={t('config_management.visual.sections.remote.description')}
              >
                <SectionStack>
                  <SectionGrid>
                    <ToggleRow
                      title={t('config_management.visual.sections.remote.allow_remote')}
                      description={t('config_management.visual.sections.remote.allow_remote_desc')}
                      checked={values.rmAllowRemote}
                      disabled={disabled}
                      onChange={(rmAllowRemote) => onChange({ rmAllowRemote })}
                    />
                    <ToggleRow
                      title={t('config_management.visual.sections.remote.disable_panel')}
                      description={t('config_management.visual.sections.remote.disable_panel_desc')}
                      checked={values.rmDisableControlPanel}
                      disabled={disabled}
                      onChange={(rmDisableControlPanel) => onChange({ rmDisableControlPanel })}
                    />
                    <ToggleRow
                      title={t(
                        'config_management.visual.sections.remote.disable_auto_update_panel'
                      )}
                      description={t(
                        'config_management.visual.sections.remote.disable_auto_update_panel_desc'
                      )}
                      checked={values.rmDisableAutoUpdatePanel}
                      disabled={disabled}
                      onChange={(rmDisableAutoUpdatePanel) =>
                        onChange({ rmDisableAutoUpdatePanel })
                      }
                    />
                  </SectionGrid>
                  <SectionGrid>
                    <Input
                      label={t('config_management.visual.sections.remote.secret_key')}
                      type="password"
                      placeholder={t(
                        'config_management.visual.sections.remote.secret_key_placeholder'
                      )}
                      value={values.rmSecretKey}
                      onChange={(e) => onChange({ rmSecretKey: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.remote.panel_repo')}
                      placeholder="https://github.com/therealtinhtute/llmhub"
                      value={values.rmPanelRepo}
                      onChange={(e) => onChange({ rmPanelRepo: e.target.value })}
                      disabled={disabled}
                    />
                  </SectionGrid>
                </SectionStack>
              </SectionSubsection>
            </SectionStack>
          </ConfigSection>

          <ConfigSection
            id="auth"
            ref={(node) => {
              sectionRefs.current.auth = node;
            }}
            indexLabel="02"
            icon={<IconKey size={16} />}
            title={t('config_management.visual.sections.auth.title')}
            description={t('config_management.visual.sections.auth.description')}
          >
            <SectionStack>
              <Input
                label={t('config_management.visual.sections.auth.auth_dir')}
                placeholder="~/.llmhub"
                value={values.authDir}
                onChange={(e) => onChange({ authDir: e.target.value })}
                disabled={disabled}
                hint={t('config_management.visual.sections.auth.auth_dir_hint')}
              />
              <div className="flex flex-col gap-3 p-4 border border-border bg-transparent">
                <ApiKeysCardEditor
                  value={values.apiKeysText}
                  disabled={disabled}
                  onChange={handleApiKeysTextChange}
                />
              </div>
            </SectionStack>
          </ConfigSection>

          <ConfigSection
            id="system"
            ref={(node) => {
              sectionRefs.current.system = node;
            }}
            indexLabel="03"
            icon={<IconDiamond size={16} />}
            title={t('config_management.visual.sections.system.title')}
            description={t('config_management.visual.sections.system.description')}
          >
            <SectionStack>
              <SectionGrid>
                <ToggleRow
                  title={t('config_management.visual.sections.system.debug')}
                  description={t('config_management.visual.sections.system.debug_desc')}
                  checked={values.debug}
                  disabled={disabled}
                  onChange={(debug) => onChange({ debug })}
                />
                <ToggleRow
                  title={t('config_management.visual.sections.system.commercial_mode')}
                  description={t('config_management.visual.sections.system.commercial_mode_desc')}
                  checked={values.commercialMode}
                  disabled={disabled}
                  onChange={(commercialMode) => onChange({ commercialMode })}
                />
                <ToggleRow
                  title={t('config_management.visual.sections.system.logging_to_file')}
                  description={t('config_management.visual.sections.system.logging_to_file_desc')}
                  checked={values.loggingToFile}
                  disabled={disabled}
                  onChange={(loggingToFile) => onChange({ loggingToFile })}
                />
              </SectionGrid>

              <SectionGrid>
                <Input
                  label={t('config_management.visual.sections.system.logs_max_size')}
                  type="number"
                  placeholder="0"
                  value={values.logsMaxTotalSizeMb}
                  onChange={(e) => onChange({ logsMaxTotalSizeMb: e.target.value })}
                  disabled={disabled}
                  error={logsMaxSizeError}
                />
                <Input
                  label={t('config_management.visual.sections.system.error_logs_max_files')}
                  type="number"
                  placeholder="10"
                  value={values.errorLogsMaxFiles}
                  onChange={(e) => onChange({ errorLogsMaxFiles: e.target.value })}
                  disabled={disabled}
                  error={errorLogsMaxFilesError}
                />
                <Input
                  label={t('config_management.visual.sections.system.redis_usage_retention')}
                  type="number"
                  placeholder="60"
                  value={values.redisUsageQueueRetentionSeconds}
                  onChange={(e) => onChange({ redisUsageQueueRetentionSeconds: e.target.value })}
                  disabled={disabled}
                  hint={t('config_management.visual.sections.system.redis_usage_retention_hint')}
                  error={redisUsageQueueRetentionError}
                />
              </SectionGrid>
              <SectionGrid>
                <ToggleRow
                  title={t('config_management.visual.sections.system.usage_statistics_enabled')}
                  description={t(
                    'config_management.visual.sections.system.usage_statistics_enabled_desc'
                  )}
                  checked={values.usageStatisticsEnabled}
                  disabled={disabled}
                  onChange={(usageStatisticsEnabled) => onChange({ usageStatisticsEnabled })}
                />
                <ToggleRow
                  title={t('config_management.visual.sections.system.antigravity_signature_cache')}
                  description={t(
                    'config_management.visual.sections.system.antigravity_signature_cache_desc'
                  )}
                  checked={values.antigravitySignatureCacheEnabled}
                  disabled={disabled}
                  onChange={(antigravitySignatureCacheEnabled) =>
                    onChange({ antigravitySignatureCacheEnabled })
                  }
                />
                <ToggleRow
                  title={t('config_management.visual.sections.system.antigravity_signature_strict')}
                  description={t(
                    'config_management.visual.sections.system.antigravity_signature_strict_desc'
                  )}
                  checked={values.antigravitySignatureBypassStrict}
                  disabled={disabled}
                  onChange={(antigravitySignatureBypassStrict) =>
                    onChange({ antigravitySignatureBypassStrict })
                  }
                />
              </SectionGrid>

              <SectionSubsection
                title={t('config_management.visual.sections.headers.title')}
                description={t('config_management.visual.sections.headers.description')}
              >
                <SectionStack>
                  <div className="flex flex-col gap-[5px]">
                    <h3 className="m-0 text-foreground text-[15px] font-bold leading-tight">
                      {t('config_management.visual.sections.headers.claude_title')}
                    </h3>
                  </div>
                  <SectionGrid>
                    <Input
                      label={t('config_management.visual.sections.headers.user_agent')}
                      placeholder="claude-cli/2.1.44 (external, sdk-cli)"
                      value={values.claudeHeaderUserAgent}
                      onChange={(e) => onChange({ claudeHeaderUserAgent: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.headers.package_version')}
                      placeholder="0.74.0"
                      value={values.claudeHeaderPackageVersion}
                      onChange={(e) => onChange({ claudeHeaderPackageVersion: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.headers.runtime_version')}
                      placeholder="v24.3.0"
                      value={values.claudeHeaderRuntimeVersion}
                      onChange={(e) => onChange({ claudeHeaderRuntimeVersion: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.headers.os')}
                      placeholder="MacOS"
                      value={values.claudeHeaderOs}
                      onChange={(e) => onChange({ claudeHeaderOs: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.headers.arch')}
                      placeholder="arm64"
                      value={values.claudeHeaderArch}
                      onChange={(e) => onChange({ claudeHeaderArch: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.headers.timeout')}
                      placeholder="600"
                      value={values.claudeHeaderTimeout}
                      onChange={(e) => onChange({ claudeHeaderTimeout: e.target.value })}
                      disabled={disabled}
                    />
                  </SectionGrid>
                  <SectionGrid>
                    <ToggleRow
                      title={t('config_management.visual.sections.headers.stabilize_device')}
                      description={t(
                        'config_management.visual.sections.headers.stabilize_device_desc'
                      )}
                      checked={values.claudeHeaderStabilizeDeviceProfile}
                      disabled={disabled}
                      onChange={(claudeHeaderStabilizeDeviceProfile) =>
                        onChange({ claudeHeaderStabilizeDeviceProfile })
                      }
                    />
                  </SectionGrid>
                  <Divider />
                  <div className="flex flex-col gap-[5px]">
                    <h3 className="m-0 text-foreground text-[15px] font-bold leading-tight">
                      {t('config_management.visual.sections.headers.codex_title')}
                    </h3>
                  </div>
                  <SectionGrid>
                    <Input
                      label={t('config_management.visual.sections.headers.user_agent')}
                      placeholder="codex_cli_rs/0.114.0 (Mac OS 14.2.0; x86_64) vscode/1.111.0"
                      value={values.codexHeaderUserAgent}
                      onChange={(e) => onChange({ codexHeaderUserAgent: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.headers.beta_features')}
                      placeholder="multi_agent"
                      value={values.codexHeaderBetaFeatures}
                      onChange={(e) => onChange({ codexHeaderBetaFeatures: e.target.value })}
                      disabled={disabled}
                    />
                  </SectionGrid>
                </SectionStack>
              </SectionSubsection>

              <SectionSubsection
                title={t('config_management.visual.sections.network.title')}
                description={t('config_management.visual.sections.network.description')}
              >
                <SectionStack>
                  <SectionGrid>
                    <Input
                      label={t('config_management.visual.sections.network.proxy_url')}
                      placeholder="socks5://user:pass@127.0.0.1:1080/"
                      value={values.proxyUrl}
                      onChange={(e) => onChange({ proxyUrl: e.target.value })}
                      disabled={disabled}
                    />
                    <Input
                      label={t('config_management.visual.sections.network.request_retry')}
                      type="number"
                      placeholder="3"
                      value={values.requestRetry}
                      onChange={(e) => onChange({ requestRetry: e.target.value })}
                      disabled={disabled}
                      error={requestRetryError}
                    />
                    <Input
                      label={t('config_management.visual.sections.network.max_retry_credentials')}
                      type="number"
                      placeholder="0"
                      value={values.maxRetryCredentials}
                      onChange={(e) => onChange({ maxRetryCredentials: e.target.value })}
                      disabled={disabled}
                      hint={t(
                        'config_management.visual.sections.network.max_retry_credentials_hint'
                      )}
                      error={maxRetryCredentialsError}
                    />
                    <Input
                      label={t('config_management.visual.sections.network.max_retry_interval')}
                      type="number"
                      placeholder="30"
                      value={values.maxRetryInterval}
                      onChange={(e) => onChange({ maxRetryInterval: e.target.value })}
                      disabled={disabled}
                      error={maxRetryIntervalError}
                    />
                    <Input
                      label={t(
                        'config_management.visual.sections.network.auth_auto_refresh_workers'
                      )}
                      type="number"
                      placeholder="16"
                      value={values.authAutoRefreshWorkers}
                      onChange={(e) => onChange({ authAutoRefreshWorkers: e.target.value })}
                      disabled={disabled}
                      hint={t(
                        'config_management.visual.sections.network.auth_auto_refresh_workers_hint'
                      )}
                      error={authAutoRefreshWorkersError}
                    />
                    <FieldShell
                      label={t('config_management.visual.sections.network.routing_strategy')}
                      labelId={routingStrategyLabelId}
                      hint={t('config_management.visual.sections.network.routing_strategy_hint')}
                      hintId={routingStrategyHintId}
                    >
                      <Select
                        value={values.routingStrategy}
                        options={[
                          {
                            value: 'round-robin',
                            label: t(
                              'config_management.visual.sections.network.strategy_round_robin'
                            ),
                          },
                          {
                            value: 'fill-first',
                            label: t(
                              'config_management.visual.sections.network.strategy_fill_first'
                            ),
                          },
                        ]}
                        id={`${routingStrategyLabelId}-select`}
                        disabled={disabled}
                        ariaLabelledBy={routingStrategyLabelId}
                        ariaDescribedBy={routingStrategyHintId}
                        onChange={(nextValue) =>
                          onChange({
                            routingStrategy: nextValue as VisualConfigValues['routingStrategy'],
                          })
                        }
                      />
                    </FieldShell>
                    <FieldShell
                      label={t(
                        'config_management.visual.sections.network.disable_image_generation'
                      )}
                      labelId={disableImageGenerationLabelId}
                      hint={t(
                        'config_management.visual.sections.network.disable_image_generation_hint'
                      )}
                      hintId={disableImageGenerationHintId}
                    >
                      <Select
                        value={values.disableImageGeneration}
                        options={disableImageGenerationOptions}
                        id={`${disableImageGenerationLabelId}-select`}
                        disabled={disabled}
                        ariaLabelledBy={disableImageGenerationLabelId}
                        ariaDescribedBy={disableImageGenerationHintId}
                        onChange={(nextValue) =>
                          onChange({
                            disableImageGeneration:
                              nextValue as VisualConfigValues['disableImageGeneration'],
                          })
                        }
                      />
                    </FieldShell>
                    <Input
                      label={t('config_management.visual.sections.network.session_affinity_ttl')}
                      placeholder="1h"
                      value={values.routingSessionAffinityTTL}
                      onChange={(e) => onChange({ routingSessionAffinityTTL: e.target.value })}
                      disabled={disabled}
                    />
                  </SectionGrid>

                  <SectionGrid>
                    <ToggleRow
                      title={t('config_management.visual.sections.network.force_model_prefix')}
                      description={t(
                        'config_management.visual.sections.network.force_model_prefix_desc'
                      )}
                      checked={values.forceModelPrefix}
                      disabled={disabled}
                      onChange={(forceModelPrefix) => onChange({ forceModelPrefix })}
                    />
                    <ToggleRow
                      title={t('config_management.visual.sections.network.passthrough_headers')}
                      description={t(
                        'config_management.visual.sections.network.passthrough_headers_desc'
                      )}
                      checked={values.passthroughHeaders}
                      disabled={disabled}
                      onChange={(passthroughHeaders) => onChange({ passthroughHeaders })}
                    />
                    <ToggleRow
                      title={t('config_management.visual.sections.network.disable_cooling')}
                      description={t(
                        'config_management.visual.sections.network.disable_cooling_desc'
                      )}
                      checked={values.disableCooling}
                      disabled={disabled}
                      onChange={(disableCooling) => onChange({ disableCooling })}
                    />
                    <ToggleRow
                      title={t('config_management.visual.sections.network.session_affinity')}
                      checked={values.routingSessionAffinity}
                      disabled={disabled}
                      onChange={(routingSessionAffinity) => onChange({ routingSessionAffinity })}
                    />
                    <ToggleRow
                      title={t('config_management.visual.sections.network.ws_auth')}
                      description={t('config_management.visual.sections.network.ws_auth_desc')}
                      checked={values.wsAuth}
                      disabled={disabled}
                      onChange={(wsAuth) => onChange({ wsAuth })}
                    />
                    <ToggleRow
                      title={t(
                        'config_management.visual.sections.network.enable_gemini_cli_endpoint'
                      )}
                      description={t(
                        'config_management.visual.sections.network.enable_gemini_cli_endpoint_desc'
                      )}
                      checked={values.enableGeminiCliEndpoint}
                      disabled={disabled}
                      onChange={(enableGeminiCliEndpoint) => onChange({ enableGeminiCliEndpoint })}
                    />
                  </SectionGrid>
                </SectionStack>
              </SectionSubsection>
            </SectionStack>
          </ConfigSection>

          <ConfigSection
            id="quota"
            ref={(node) => {
              sectionRefs.current.quota = node;
            }}
            indexLabel="04"
            icon={<IconTimer size={16} />}
            title={t('config_management.visual.sections.quota.title')}
            description={t('config_management.visual.sections.quota.description')}
          >
            <SectionGrid>
              <ToggleRow
                title={t('config_management.visual.sections.quota.switch_project')}
                description={t('config_management.visual.sections.quota.switch_project_desc')}
                checked={values.quotaSwitchProject}
                disabled={disabled}
                onChange={(quotaSwitchProject) => onChange({ quotaSwitchProject })}
              />
              <ToggleRow
                title={t('config_management.visual.sections.quota.switch_preview_model')}
                description={t('config_management.visual.sections.quota.switch_preview_model_desc')}
                checked={values.quotaSwitchPreviewModel}
                disabled={disabled}
                onChange={(quotaSwitchPreviewModel) => onChange({ quotaSwitchPreviewModel })}
              />
              <ToggleRow
                title={t('config_management.visual.sections.quota.antigravity_credits')}
                checked={values.quotaAntigravityCredits}
                disabled={disabled}
                onChange={(quotaAntigravityCredits) => onChange({ quotaAntigravityCredits })}
              />
            </SectionGrid>
          </ConfigSection>

          <ConfigSection
            id="streaming"
            ref={(node) => {
              sectionRefs.current.streaming = node;
            }}
            indexLabel="05"
            icon={<IconSatellite size={16} />}
            title={t('config_management.visual.sections.streaming.title')}
            description={t('config_management.visual.sections.streaming.description')}
          >
            <SectionStack>
              <SectionGrid>
                <FieldShell
                  label={t('config_management.visual.sections.streaming.keepalive_seconds')}
                  htmlFor={keepaliveInputId}
                  hint={t('config_management.visual.sections.streaming.keepalive_hint')}
                  hintId={keepaliveHintId}
                  error={keepaliveError}
                  errorId={keepaliveErrorId}
                >
                  <div className="relative">
                    <BaseInput
                      id={keepaliveInputId}
                      type="number"
                      placeholder="0"
                      value={values.streaming.keepaliveSeconds}
                      onChange={(e) =>
                        onChange({
                          streaming: {
                            ...values.streaming,
                            keepaliveSeconds: e.target.value,
                          },
                        })
                      }
                      disabled={disabled}
                    />
                    {isKeepaliveDisabled ? (
                      <span className="absolute right-2 top-1/2 -translate-y-1/2 inline-flex items-center min-h-[24px] px-2 border border-border bg-background text-muted-foreground text-[11px] font-bold">
                        {t('config_management.visual.sections.streaming.disabled')}
                      </span>
                    ) : null}
                  </div>
                </FieldShell>

                <Input
                  label={t('config_management.visual.sections.streaming.bootstrap_retries')}
                  type="number"
                  placeholder="1"
                  value={values.streaming.bootstrapRetries}
                  onChange={(e) =>
                    onChange({
                      streaming: {
                        ...values.streaming,
                        bootstrapRetries: e.target.value,
                      },
                    })
                  }
                  disabled={disabled}
                  hint={t('config_management.visual.sections.streaming.bootstrap_hint')}
                  error={bootstrapRetriesError}
                />
              </SectionGrid>

              <SectionGrid>
                <FieldShell
                  label={t('config_management.visual.sections.streaming.nonstream_keepalive')}
                  htmlFor={nonstreamKeepaliveInputId}
                  hint={t('config_management.visual.sections.streaming.nonstream_keepalive_hint')}
                  hintId={nonstreamKeepaliveHintId}
                  error={nonstreamKeepaliveError}
                  errorId={nonstreamKeepaliveErrorId}
                >
                  <div className="relative">
                    <BaseInput
                      id={nonstreamKeepaliveInputId}
                      type="number"
                      placeholder="0"
                      value={values.streaming.nonstreamKeepaliveInterval}
                      onChange={(e) =>
                        onChange({
                          streaming: {
                            ...values.streaming,
                            nonstreamKeepaliveInterval: e.target.value,
                          },
                        })
                      }
                      disabled={disabled}
                    />
                    {isNonstreamKeepaliveDisabled ? (
                      <span className="absolute right-2 top-1/2 -translate-y-1/2 inline-flex items-center min-h-[24px] px-2 border border-border bg-background text-muted-foreground text-[11px] font-bold">
                        {t('config_management.visual.sections.streaming.disabled')}
                      </span>
                    ) : null}
                  </div>
                </FieldShell>
              </SectionGrid>
            </SectionStack>
          </ConfigSection>

          <ConfigSection
            id="payload"
            ref={(node) => {
              sectionRefs.current.payload = node;
            }}
            indexLabel="06"
            icon={<IconCode size={16} />}
            title={t('config_management.visual.sections.payload.title')}
            description={t('config_management.visual.sections.payload.description')}
          >
            <SectionStack>
              <SectionSubsection
                title={t('config_management.visual.sections.payload.default_rules')}
                description={t('config_management.visual.sections.payload.default_rules_desc')}
              >
                <PayloadRulesEditor
                  value={values.payloadDefaultRules}
                  disabled={disabled}
                  onChange={handlePayloadDefaultRulesChange}
                />
              </SectionSubsection>

              <SectionSubsection
                title={t('config_management.visual.sections.payload.default_raw_rules')}
                description={t('config_management.visual.sections.payload.default_raw_rules_desc')}
              >
                <PayloadRulesEditor
                  value={values.payloadDefaultRawRules}
                  disabled={disabled}
                  rawJsonValues
                  onChange={handlePayloadDefaultRawRulesChange}
                />
              </SectionSubsection>

              <SectionSubsection
                title={t('config_management.visual.sections.payload.override_rules')}
                description={t('config_management.visual.sections.payload.override_rules_desc')}
              >
                <PayloadRulesEditor
                  value={values.payloadOverrideRules}
                  disabled={disabled}
                  protocolFirst
                  onChange={handlePayloadOverrideRulesChange}
                />
              </SectionSubsection>

              <SectionSubsection
                title={t('config_management.visual.sections.payload.override_raw_rules')}
                description={t('config_management.visual.sections.payload.override_raw_rules_desc')}
              >
                <PayloadRulesEditor
                  value={values.payloadOverrideRawRules}
                  disabled={disabled}
                  protocolFirst
                  rawJsonValues
                  onChange={handlePayloadOverrideRawRulesChange}
                />
              </SectionSubsection>

              <SectionSubsection
                title={t('config_management.visual.sections.payload.filter_rules')}
                description={t('config_management.visual.sections.payload.filter_rules_desc')}
              >
                <PayloadFilterRulesEditor
                  value={values.payloadFilterRules}
                  disabled={disabled}
                  onChange={handlePayloadFilterRulesChange}
                />
              </SectionSubsection>
            </SectionStack>
          </ConfigSection>
        </div>
      </div>
    </div>
  );
}
