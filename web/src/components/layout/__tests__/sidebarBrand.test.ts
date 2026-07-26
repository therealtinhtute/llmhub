import { describe, expect, it } from 'vitest';
import en from '@/i18n/locales/en.json';
import vi from '@/i18n/locales/vi.json';
import mainLayoutSource from '../MainLayout.tsx?raw';
import indexHtml from '../../../../index.html?raw';

describe('sidebar brand', () => {
  it('renders the enlarged logo above the lowercase italic serif brand label', () => {
    expect(mainLayoutSource).toContain('grid justify-items-start gap-1');
    expect(mainLayoutSource).toContain('size-15 object-contain');
    expect(mainLayoutSource).toContain(
      'truncate font-serif text-xl font-semibold italic lowercase text-sidebar-foreground'
    );
    expect(mainLayoutSource).toContain("{t('title.abbr')}");
    expect(mainLayoutSource.indexOf('<img')).toBeLessThan(mainLayoutSource.indexOf('<span'));
  });

  it('uses the canonical brand casing in every locale', () => {
    expect(en.title.abbr).toBe('LLMHub');
    expect(vi.title.abbr).toBe('LLMHub');
  });

  it('loads the required Lora italic font weight', () => {
    expect(indexHtml).toContain('family=Lora:ital,wght@1,600');
  });
});
