'use client';

import { useMemo } from 'react';
import { HeatmapTile, HeatmapData } from '@/lib/api/heatmap';
import { useRouter } from 'next/navigation';
import { themeColors } from '@/lib/theme';
import { useTheme } from '@/lib/contexts/ThemeContext';

// ─── Deterministic sparkline RNG ─────────────────────────────────

function seededRandom(symbol: string) {
  let seed = 0;
  for (let i = 0; i < symbol.length; i++) {
    seed = ((seed << 5) - seed + symbol.charCodeAt(i)) | 0;
  }
  return () => {
    seed = (seed * 16807) % 2147483647;
    return (seed & 0x7fffffff) / 2147483647;
  };
}

// ─── Mini Sparkline ──────────────────────────────────────────────

function MiniSparkline({
  symbol,
  changePct,
  width = 80,
  height = 32,
}: {
  symbol: string;
  changePct: number;
  width?: number;
  height?: number;
}) {
  const color = changePct >= 0 ? themeColors.accent.positive : themeColors.accent.negative;
  const rand = seededRandom(symbol);
  const pad = 4;
  const w = width - pad * 2;
  const h = height - pad * 2;
  const numPoints = 7;

  const points = Array.from({ length: numPoints }, (_, i) => {
    const t = i / (numPoints - 1);
    // Trend from center toward top (positive) or bottom (negative)
    const endY = changePct >= 0 ? 0.2 : 0.8;
    const trendY = 0.5 + (endY - 0.5) * t;
    const noise = (rand() - 0.5) * 0.25;
    const y = Math.max(0.05, Math.min(0.95, trendY + noise));
    return `${pad + t * w},${pad + y * h}`;
  }).join(' ');

  return (
    <svg width={width} height={height} className="flex-shrink-0">
      <polyline
        points={points}
        fill="none"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

// ─── Change Badge ────────────────────────────────────────────────

function ChangeBadge({ value, size = 'md' }: { value: number; size?: 'sm' | 'md' | 'lg' }) {
  const isPositive = value >= 0;
  const sizeClasses = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-sm px-3 py-1',
    lg: 'text-lg px-4 py-1.5 font-bold',
  };

  return (
    <span
      className={`inline-flex items-center rounded-full font-semibold ${sizeClasses[size]}`}
      style={{
        color: isPositive ? themeColors.accent.positive : themeColors.accent.negative,
        backgroundColor: isPositive ? themeColors.accent.positiveBg : themeColors.accent.negativeBg,
      }}
    >
      {isPositive ? '+' : ''}
      {value.toFixed(2)}%
    </span>
  );
}

// ─── Format helpers ──────────────────────────────────────────────

function formatPrice(price: number): string {
  return `$${price.toFixed(2)}`;
}

function formatVolume(vol: number): string {
  if (vol >= 1e9) return `${(vol / 1e9).toFixed(1)}B`;
  if (vol >= 1e6) return `${(vol / 1e6).toFixed(1)}M`;
  if (vol >= 1e3) return `${(vol / 1e3).toFixed(1)}K`;
  return vol.toString();
}

function formatMarketCap(cap: number): string {
  if (cap >= 1e12) return `$${(cap / 1e12).toFixed(2)}T`;
  if (cap >= 1e9) return `$${(cap / 1e9).toFixed(2)}B`;
  if (cap >= 1e6) return `$${(cap / 1e6).toFixed(1)}M`;
  return `$${cap.toLocaleString()}`;
}

// ─── Hero View (1–3 stocks) ─────────────────────────────────────

export function HeatmapHeroView({
  data,
  onTileClick,
}: {
  data: HeatmapData;
  onTileClick?: (symbol: string) => void;
}) {
  const router = useRouter();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';

  const handleClick = (symbol: string) => {
    if (onTileClick) onTileClick(symbol);
    else router.push(`/ticker/${symbol}`);
  };

  return (
    <div className="space-y-4">
      {data.tiles.map((tile) => {
        const isPositive = tile.price_change_pct >= 0;
        const tintBg = isPositive
          ? isDark
            ? 'rgba(52, 211, 153, 0.05)'
            : 'rgba(52, 211, 153, 0.04)'
          : isDark
            ? 'rgba(248, 113, 113, 0.05)'
            : 'rgba(248, 113, 113, 0.04)';
        const tintBorder = isPositive
          ? isDark
            ? 'rgba(52, 211, 153, 0.15)'
            : 'rgba(52, 211, 153, 0.12)'
          : isDark
            ? 'rgba(248, 113, 113, 0.15)'
            : 'rgba(248, 113, 113, 0.12)';

        return (
          <div
            key={tile.symbol}
            onClick={() => handleClick(tile.symbol)}
            className="rounded-xl border p-6 cursor-pointer transition-all hover:shadow-lg"
            style={{
              backgroundColor: tintBg,
              borderColor: tintBorder,
              minHeight: '180px',
            }}
          >
            <div className="flex items-center justify-between h-full">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <span className="text-3xl font-bold text-ic-text-primary">{tile.symbol}</span>
                  <ChangeBadge value={tile.price_change_pct} size="lg" />
                </div>
                <div className="text-lg text-ic-text-secondary mb-4">{tile.name}</div>
                <div className="flex items-center gap-6 flex-wrap">
                  <div>
                    <div className="text-sm text-ic-text-dim">Price</div>
                    <div className="text-2xl font-semibold text-ic-text-primary">
                      {formatPrice(tile.current_price)}
                    </div>
                  </div>
                  <div>
                    <div className="text-sm text-ic-text-dim">Change</div>
                    <div
                      className="text-xl font-medium"
                      style={{
                        color: isPositive
                          ? themeColors.accent.positive
                          : themeColors.accent.negative,
                      }}
                    >
                      {isPositive ? '+' : ''}
                      {tile.price_change.toFixed(2)}
                    </div>
                  </div>
                  {tile.market_cap != null && tile.market_cap > 0 && (
                    <div>
                      <div className="text-sm text-ic-text-dim">Market Cap</div>
                      <div className="text-lg text-ic-text-secondary">
                        {formatMarketCap(tile.market_cap)}
                      </div>
                    </div>
                  )}
                  {tile.volume != null && tile.volume > 0 && (
                    <div>
                      <div className="text-sm text-ic-text-dim">Volume</div>
                      <div className="text-lg text-ic-text-secondary">
                        {formatVolume(tile.volume)}
                      </div>
                    </div>
                  )}
                </div>
              </div>
              <div className="ml-8 hidden sm:block">
                <MiniSparkline
                  symbol={tile.symbol}
                  changePct={tile.price_change_pct}
                  width={160}
                  height={80}
                />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─── Card Grid View (4–12 stocks) ───────────────────────────────

export function HeatmapCardGrid({
  data,
  onTileClick,
}: {
  data: HeatmapData;
  onTileClick?: (symbol: string) => void;
}) {
  const router = useRouter();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === 'dark';

  const handleClick = (symbol: string) => {
    if (onTileClick) onTileClick(symbol);
    else router.push(`/ticker/${symbol}`);
  };

  const sortedTiles = useMemo(
    () =>
      [...data.tiles].sort((a, b) => Math.abs(b.price_change_pct) - Math.abs(a.price_change_pct)),
    [data.tiles]
  );

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {sortedTiles.map((tile) => {
        const isPositive = tile.price_change_pct >= 0;
        const tintBg = isPositive
          ? isDark
            ? 'rgba(52, 211, 153, 0.04)'
            : 'rgba(52, 211, 153, 0.03)'
          : isDark
            ? 'rgba(248, 113, 113, 0.04)'
            : 'rgba(248, 113, 113, 0.03)';
        const tintBorder = isPositive
          ? isDark
            ? 'rgba(52, 211, 153, 0.12)'
            : 'rgba(52, 211, 153, 0.1)'
          : isDark
            ? 'rgba(248, 113, 113, 0.12)'
            : 'rgba(248, 113, 113, 0.1)';

        return (
          <div
            key={tile.symbol}
            onClick={() => handleClick(tile.symbol)}
            className="rounded-lg border p-4 cursor-pointer transition-all hover:shadow-md"
            style={{ backgroundColor: tintBg, borderColor: tintBorder }}
          >
            <div className="flex items-start justify-between mb-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-lg font-bold text-ic-text-primary">{tile.symbol}</span>
                  <ChangeBadge value={tile.price_change_pct} size="md" />
                </div>
                <div className="text-sm text-ic-text-dim truncate mt-0.5">{tile.name}</div>
              </div>
              <MiniSparkline
                symbol={tile.symbol}
                changePct={tile.price_change_pct}
                width={72}
                height={28}
              />
            </div>
            <div className="flex items-end justify-between">
              <div className="text-xl font-semibold text-ic-text-primary">
                {formatPrice(tile.current_price)}
              </div>
              <div className="text-xs text-ic-text-dim">
                {tile.market_cap != null && tile.market_cap > 0
                  ? formatMarketCap(tile.market_cap)
                  : tile.volume != null && tile.volume > 0
                    ? `Vol: ${formatVolume(tile.volume)}`
                    : ''}
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─── Bar Chart View ─────────────────────────────────────────────

export function HeatmapBarChart({
  data,
  onTileClick,
  maxHeight,
}: {
  data: HeatmapData;
  onTileClick?: (symbol: string) => void;
  maxHeight?: number;
}) {
  const router = useRouter();

  const handleClick = (symbol: string) => {
    if (onTileClick) onTileClick(symbol);
    else router.push(`/ticker/${symbol}`);
  };

  const sorted = useMemo(
    () => [...data.tiles].sort((a, b) => b.price_change_pct - a.price_change_pct),
    [data.tiles]
  );

  const maxAbsChange = Math.max(...sorted.map((t) => Math.abs(t.price_change_pct)), 0.01);

  return (
    <div className="space-y-1.5 overflow-y-auto" style={maxHeight ? { maxHeight } : undefined}>
      {sorted.map((tile) => {
        const isPositive = tile.price_change_pct >= 0;
        const barWidth = Math.max(2, (Math.abs(tile.price_change_pct) / maxAbsChange) * 100);
        const barColor = isPositive ? themeColors.accent.positive : themeColors.accent.negative;
        const barBg = isPositive ? themeColors.accent.positiveBg : themeColors.accent.negativeBg;

        return (
          <div
            key={tile.symbol}
            onClick={() => handleClick(tile.symbol)}
            className="flex items-center gap-3 px-3 py-2 rounded-md cursor-pointer hover:bg-ic-surface-hover transition-colors"
          >
            <span className="w-16 text-sm font-bold text-ic-text-primary flex-shrink-0">
              {tile.symbol}
            </span>
            <div
              className="flex-1 h-6 rounded-full overflow-hidden relative"
              style={{ backgroundColor: barBg }}
            >
              <div
                className="h-full rounded-full transition-all"
                style={{
                  width: `${barWidth}%`,
                  backgroundColor: barColor,
                  opacity: 0.7,
                }}
              />
            </div>
            <span
              className="w-20 text-right text-sm font-semibold flex-shrink-0"
              style={{ color: barColor }}
            >
              {isPositive ? '+' : ''}
              {tile.price_change_pct.toFixed(2)}%
            </span>
          </div>
        );
      })}
    </div>
  );
}

// ─── View mode helpers ──────────────────────────────────────────

export type ViewMode = 'auto' | 'treemap' | 'cards' | 'bars';

export type EffectiveView = 'hero' | 'cards' | 'hybrid' | 'treemap' | 'bars';

export function getEffectiveView(viewMode: ViewMode, tileCount: number): EffectiveView {
  if (viewMode === 'treemap') return 'treemap';
  if (viewMode === 'bars') return 'bars';
  if (viewMode === 'cards') return tileCount <= 3 ? 'hero' : 'cards';

  // Auto mode
  if (tileCount <= 3) return 'hero';
  if (tileCount <= 12) return 'cards';
  if (tileCount <= 30) return 'hybrid';
  return 'treemap';
}
