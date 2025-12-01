// =============================================================================
// Theme Utilities
// =============================================================================
// Helper functions for using theme colors in JavaScript/TypeScript code.
// Useful for chart libraries and dynamic styling that can't use CSS variables.
// Note: These values should stay in sync with styles/theme.css
// =============================================================================

// =============================================================================
// Type Definitions
// =============================================================================

export type ThemeMode = 'light' | 'dark';

export type Rating = 'Strong Buy' | 'Buy' | 'Hold' | 'Sell' | 'Strong Sell';

export type Sentiment = 'Bullish' | 'Positive' | 'Bearish' | 'Negative' | 'Neutral';

// =============================================================================
// Theme Colors
// =============================================================================

export const themeColors = {
  dark: {
    bgPrimary: '#09090B',
    bgSecondary: '#18181B',
    bgTertiary: '#27272A',
    surface: 'rgba(255, 255, 255, 0.03)',
    surfaceHover: 'rgba(255, 255, 255, 0.06)',
    border: 'rgba(255, 255, 255, 0.1)',
    borderSubtle: 'rgba(255, 255, 255, 0.05)',
    textPrimary: '#FAFAFA',
    textSecondary: '#D4D4D8',
    textMuted: '#A1A1AA',
    textDim: '#71717A',
  },
  light: {
    bgPrimary: '#FFFFFF',
    bgSecondary: '#F4F4F5',
    bgTertiary: '#E4E4E7',
    surface: 'rgba(0, 0, 0, 0.02)',
    surfaceHover: 'rgba(0, 0, 0, 0.04)',
    border: 'rgba(0, 0, 0, 0.1)',
    borderSubtle: 'rgba(0, 0, 0, 0.05)',
    textPrimary: '#09090B',
    textSecondary: '#27272A',
    textMuted: '#52525B',
    textDim: '#71717A',
  },
  accent: {
    blue: '#3B82F6',
    blueHover: '#2563EB',
    cyan: '#22D3EE',
    positive: '#34D399',
    positiveBg: 'rgba(52, 211, 153, 0.1)',
    negative: '#F87171',
    negativeBg: 'rgba(248, 113, 113, 0.1)',
    warning: '#FBBF24',
    warningBg: 'rgba(251, 191, 36, 0.1)',
    orange: '#F97316',
    orangeBg: 'rgba(249, 115, 22, 0.15)',
  },
};

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Get theme colors for the specified mode, merged with accent colors
 */
export function getThemeColors(mode: ThemeMode) {
  return {
    ...themeColors[mode],
    ...themeColors.accent,
  };
}

/**
 * Get styling for analyst ratings
 */
export function getRatingStyle(rating: Rating): {
  color: string;
  backgroundColor: string;
  borderColor: string;
} {
  switch (rating) {
    case 'Strong Buy':
    case 'Buy':
      return {
        color: themeColors.accent.positive,
        backgroundColor: themeColors.accent.positiveBg,
        borderColor: themeColors.accent.positive,
      };
    case 'Hold':
      return {
        color: themeColors.accent.warning,
        backgroundColor: themeColors.accent.warningBg,
        borderColor: themeColors.accent.warning,
      };
    case 'Sell':
    case 'Strong Sell':
      return {
        color: themeColors.accent.negative,
        backgroundColor: themeColors.accent.negativeBg,
        borderColor: themeColors.accent.negative,
      };
    default:
      return {
        color: themeColors.dark.textMuted,
        backgroundColor: 'transparent',
        borderColor: themeColors.dark.textMuted,
      };
  }
}

/**
 * Get color based on numeric value (positive/negative/zero)
 */
export function getValueColor(value: number): string {
  if (value > 0) {
    return themeColors.accent.positive; // #34D399 (green)
  } else if (value < 0) {
    return themeColors.accent.negative; // #F87171 (red)
  }
  return themeColors.dark.textMuted; // #A1A1AA (gray/neutral)
}

/**
 * Get color based on sentiment
 */
export function getSentimentColor(sentiment: Sentiment | string): string {
  switch (sentiment) {
    case 'Bullish':
    case 'Positive':
      return themeColors.accent.positive; // #34D399
    case 'Bearish':
    case 'Negative':
      return themeColors.accent.negative; // #F87171
    case 'Neutral':
    default:
      return themeColors.dark.textMuted; // #A1A1AA
  }
}

/**
 * Get colors optimized for chart libraries
 */
export function getChartColors(mode: ThemeMode) {
  return {
    line: themeColors.accent.blue, // '#3B82F6' - Primary chart line
    grid: mode === 'dark' ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.05)',
    text: themeColors[mode].textMuted,
    background: themeColors[mode].bgPrimary,
    positive: themeColors.accent.positive, // '#34D399'
    negative: themeColors.accent.negative, // '#F87171'
  };
}
