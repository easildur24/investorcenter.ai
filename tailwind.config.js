/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './pages/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        // Existing primary colors
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
          900: '#1e3a8a',
        },

        // Backgrounds
        'ic-bg': {
          primary: 'var(--ic-bg-primary)',
          secondary: 'var(--ic-bg-secondary)',
          tertiary: 'var(--ic-bg-tertiary)',
        },

        // Surfaces
        'ic-surface': {
          DEFAULT: 'var(--ic-surface)',
          hover: 'var(--ic-surface-hover)',
        },

        // Borders
        'ic-border': {
          DEFAULT: 'var(--ic-border)',
          subtle: 'var(--ic-border-subtle)',
        },

        // Text
        'ic-text': {
          primary: 'var(--ic-text-primary)',
          secondary: 'var(--ic-text-secondary)',
          muted: 'var(--ic-text-muted)',
          dim: 'var(--ic-text-dim)',
        },

        // Header
        'ic-header': {
          bg: 'var(--ic-header-bg)',
        },

        // Input fields
        'ic-input': {
          bg: 'var(--ic-input-bg)',
          border: 'var(--ic-input-border)',
        },

        // Accent colors
        'ic-blue': {
          DEFAULT: 'var(--ic-blue)',
          hover: 'var(--ic-blue-hover)',
        },
        'ic-cyan': 'var(--ic-cyan)',
        'ic-positive': {
          DEFAULT: 'var(--ic-positive)',
          bg: 'var(--ic-positive-bg)',
        },
        'ic-negative': {
          DEFAULT: 'var(--ic-negative)',
          bg: 'var(--ic-negative-bg)',
        },
        'ic-warning': {
          DEFAULT: 'var(--ic-warning)',
          bg: 'var(--ic-warning-bg)',
        },
        'ic-orange': {
          DEFAULT: 'var(--ic-orange)',
          bg: 'var(--ic-orange-bg)',
        },
      },
    },
  },
  plugins: [],
}
