import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { INLINE_LOGO_JPEG } from '@/assets/logoInline';

interface SplashScreenProps {
  onFinish: () => void;
  fadeOut?: boolean;
}

const FADE_OUT_DURATION = 400;

export function SplashScreen({ onFinish, fadeOut = false }: SplashScreenProps) {
  const { t } = useTranslation();

  useEffect(() => {
    if (!fadeOut) return;
    const finishTimer = setTimeout(() => {
      onFinish();
    }, FADE_OUT_DURATION);

    return () => {
      clearTimeout(finishTimer);
    };
  }, [fadeOut, onFinish]);

  return (
    <div
      className={[
        'fixed inset-0 z-[9999] flex items-center justify-center bg-background transition-opacity duration-[400ms] ease-out',
        fadeOut ? 'opacity-0 pointer-events-none' : 'opacity-100',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div className="flex flex-col items-center gap-3 [animation:splashEnter_0.6s_ease-out]">
        <img
          src={INLINE_LOGO_JPEG}
          alt="CPAMC"
          className="h-20 w-auto shadow-lg [animation:splashLogoPulse_1.5s_ease-in-out_infinite]"
        />
        <h1 className="text-[28px] font-extrabold text-foreground m-0 tracking-[-0.5px]">
          {t('splash.title')}
        </h1>
        <p className="text-base font-medium text-muted-foreground m-0 -mt-2">
          {t('splash.subtitle')}
        </p>
        <div className="w-[120px] h-[3px] bg-border rounded-full overflow-hidden mt-3">
          <div className="w-full h-full bg-primary rounded-full [animation:splashLoading_1.2s_ease-in-out_infinite] [transform-origin:left]" />
        </div>
      </div>
    </div>
  );
}
