import {
  themeColors,
  getThemeColors,
  getRatingStyle,
  getValueColor,
  getSentimentColor,
  getChartColors,
} from '../theme';
import type { ThemeMode, Rating, Sentiment } from '../theme';

describe('getThemeColors', () => {
  it('returns dark mode colors merged with accent colors', () => {
    const colors = getThemeColors('dark');
    // Dark-specific values
    expect(colors.bgPrimary).toBe('#09090B');
    expect(colors.textPrimary).toBe('#FAFAFA');
    // Accent values merged in
    expect(colors.blue).toBe('#3B82F6');
    expect(colors.positive).toBe('#34D399');
    expect(colors.negative).toBe('#F87171');
  });

  it('returns light mode colors merged with accent colors', () => {
    const colors = getThemeColors('light');
    // Light-specific values
    expect(colors.bgPrimary).toBe('#FFFFFF');
    expect(colors.textPrimary).toBe('#0F172A');
    // Accent values merged in
    expect(colors.blue).toBe('#3B82F6');
    expect(colors.positive).toBe('#34D399');
  });

  it('accent colors override mode colors if keys collide (they do not currently)', () => {
    const darkColors = getThemeColors('dark');
    const lightColors = getThemeColors('light');
    // Accent values should be the same regardless of mode
    expect(darkColors.blue).toBe(lightColors.blue);
    expect(darkColors.warning).toBe(lightColors.warning);
  });
});

describe('getRatingStyle', () => {
  it('returns positive style for Strong Buy', () => {
    const style = getRatingStyle('Strong Buy');
    expect(style.color).toBe(themeColors.accent.positive);
    expect(style.backgroundColor).toBe(themeColors.accent.positiveBg);
    expect(style.borderColor).toBe(themeColors.accent.positive);
  });

  it('returns positive style for Buy', () => {
    const style = getRatingStyle('Buy');
    expect(style.color).toBe(themeColors.accent.positive);
  });

  it('returns warning style for Hold', () => {
    const style = getRatingStyle('Hold');
    expect(style.color).toBe(themeColors.accent.warning);
    expect(style.backgroundColor).toBe(themeColors.accent.warningBg);
    expect(style.borderColor).toBe(themeColors.accent.warning);
  });

  it('returns negative style for Sell', () => {
    const style = getRatingStyle('Sell');
    expect(style.color).toBe(themeColors.accent.negative);
    expect(style.backgroundColor).toBe(themeColors.accent.negativeBg);
    expect(style.borderColor).toBe(themeColors.accent.negative);
  });

  it('returns negative style for Strong Sell', () => {
    const style = getRatingStyle('Strong Sell');
    expect(style.color).toBe(themeColors.accent.negative);
  });

  it('returns muted/default style for unknown rating', () => {
    const style = getRatingStyle('Unknown' as Rating);
    expect(style.color).toBe(themeColors.dark.textMuted);
    expect(style.backgroundColor).toBe('transparent');
    expect(style.borderColor).toBe(themeColors.dark.textMuted);
  });
});

describe('getValueColor', () => {
  it('returns positive color for positive value', () => {
    expect(getValueColor(10)).toBe(themeColors.accent.positive);
  });

  it('returns negative color for negative value', () => {
    expect(getValueColor(-5)).toBe(themeColors.accent.negative);
  });

  it('returns muted color for zero', () => {
    expect(getValueColor(0)).toBe(themeColors.dark.textMuted);
  });
});

describe('getSentimentColor', () => {
  it('returns positive color for Bullish', () => {
    expect(getSentimentColor('Bullish')).toBe(themeColors.accent.positive);
  });

  it('returns positive color for Positive', () => {
    expect(getSentimentColor('Positive')).toBe(themeColors.accent.positive);
  });

  it('returns negative color for Bearish', () => {
    expect(getSentimentColor('Bearish')).toBe(themeColors.accent.negative);
  });

  it('returns negative color for Negative', () => {
    expect(getSentimentColor('Negative')).toBe(themeColors.accent.negative);
  });

  it('returns muted color for Neutral', () => {
    expect(getSentimentColor('Neutral')).toBe(themeColors.dark.textMuted);
  });

  it('returns muted color for unknown sentiment string', () => {
    expect(getSentimentColor('Unknown')).toBe(themeColors.dark.textMuted);
  });
});

describe('getChartColors', () => {
  it('returns dark mode chart colors', () => {
    const colors = getChartColors('dark');
    expect(colors.line).toBe(themeColors.accent.blue);
    expect(colors.grid).toBe('rgba(255,255,255,0.05)');
    expect(colors.text).toBe(themeColors.dark.textMuted);
    expect(colors.background).toBe(themeColors.dark.bgPrimary);
    expect(colors.positive).toBe(themeColors.accent.positive);
    expect(colors.negative).toBe(themeColors.accent.negative);
  });

  it('returns light mode chart colors', () => {
    const colors = getChartColors('light');
    expect(colors.grid).toBe('rgba(0,0,0,0.05)');
    expect(colors.text).toBe(themeColors.light.textMuted);
    expect(colors.background).toBe(themeColors.light.bgPrimary);
  });
});

describe('themeColors constant', () => {
  it('has dark, light, and accent keys', () => {
    expect(themeColors).toHaveProperty('dark');
    expect(themeColors).toHaveProperty('light');
    expect(themeColors).toHaveProperty('accent');
  });

  it('dark and light both have required color keys', () => {
    const requiredKeys = [
      'bgPrimary',
      'bgSecondary',
      'bgTertiary',
      'textPrimary',
      'textSecondary',
      'textMuted',
      'textDim',
    ];
    for (const key of requiredKeys) {
      expect(themeColors.dark).toHaveProperty(key);
      expect(themeColors.light).toHaveProperty(key);
    }
  });
});
