'use client';

import { useEffect, useRef, useMemo } from 'react';
import * as d3 from 'd3';
import { HeatmapTile, HeatmapData } from '@/lib/api/heatmap';
import { useRouter } from 'next/navigation';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { themeColors } from '@/lib/theme';

// ─── Types ────────────────────────────────────────────────────

interface WatchListHeatmapProps {
  data: HeatmapData;
  width?: number;
  height?: number;
  onTileClick?: (symbol: string) => void;
}

// ─── Symmetric Color Scale (Issue 2) ──────────────────────────
// Breakpoints at ±0.5%, ±2%, ±5% anchored at 0%.
// Clamped so values beyond ±5% saturate to the extreme color.

const COLOR_BREAKPOINTS = [-5, -2, -0.5, 0, 0.5, 2, 5];

/** Labels displayed below the gradient legend (subset of breakpoints) */
const LEGEND_LABELS = COLOR_BREAKPOINTS.filter((bp) => bp === 0 || Math.abs(bp) >= 2).map((bp) => ({
  key: String(bp),
  text: `${bp > 0 ? '+' : ''}${bp}%`,
}));

/** Minimum tile count before sector grouping kicks in */
const SECTOR_GROUPING_THRESHOLD = 13;

function getSymmetricColorScale(
  scheme: string,
  neutralColor: string
): d3.ScaleLinear<string, string> {
  const green = '#16A34A';
  const greenMed = '#22C55E';
  const greenLight = '#86EFAC';
  const red = '#DC2626';
  const redMed = '#EF4444';
  const redLight = '#FCA5A5';

  switch (scheme) {
    case 'blue_red':
      return d3
        .scaleLinear<string>()
        .domain(COLOR_BREAKPOINTS)
        .range([themeColors.accent.blue, '#60A5FA', '#BFDBFE', neutralColor, redLight, redMed, red])
        .clamp(true);

    case 'heatmap': {
      const interp = d3.interpolateRdYlGn;
      const colors = COLOR_BREAKPOINTS.map((v) => interp((v + 5) / 10));
      return d3.scaleLinear<string>().domain(COLOR_BREAKPOINTS).range(colors).clamp(true);
    }

    case 'red_green':
    default:
      return d3
        .scaleLinear<string>()
        .domain(COLOR_BREAKPOINTS)
        .range([red, redMed, redLight, neutralColor, greenLight, greenMed, green])
        .clamp(true);
  }
}

function getTextColor(bgColor: string): string {
  const rgb = d3.rgb(bgColor);
  const luminance = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255;
  return luminance > 0.5 ? '#000000' : '#FFFFFF';
}

function formatVolume(vol: number): string {
  if (vol >= 1e9) return `${(vol / 1e9).toFixed(1)}B`;
  if (vol >= 1e6) return `${(vol / 1e6).toFixed(1)}M`;
  if (vol >= 1e3) return `${(vol / 1e3).toFixed(1)}K`;
  return vol.toString();
}

/** Safely create a DOM element with text content (prevents XSS in tooltips) */
function el(
  tag: string,
  className: string,
  text?: string | number,
  children?: HTMLElement[]
): HTMLElement {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text != null) node.textContent = String(text);
  if (children) children.forEach((c) => node.appendChild(c));
  return node;
}

// ─── Color Legend (Issue 4) ───────────────────────────────────
// 16px gradient bar with labels at -5%, -2%, 0%, +2%, +5%.

export function ColorLegend({ scheme, neutralColor }: { scheme: string; neutralColor: string }) {
  const colorScale = useMemo(
    () => getSymmetricColorScale(scheme, neutralColor),
    [scheme, neutralColor]
  );

  const stops = COLOR_BREAKPOINTS.map((bp) => {
    const pct = ((bp + 5) / 10) * 100;
    return `${colorScale(bp)} ${pct.toFixed(1)}%`;
  }).join(', ');

  return (
    <div className="mt-2">
      <div
        className="w-full rounded-sm"
        style={{
          background: `linear-gradient(to right, ${stops})`,
          height: 16,
        }}
      />
      <div className="flex justify-between mt-1">
        {LEGEND_LABELS.map((l) => (
          <span key={l.key} className="text-xs text-ic-text-dim">
            {l.text}
          </span>
        ))}
      </div>
    </div>
  );
}

// ─── Main Treemap Component ──────────────────────────────────

export default function WatchListHeatmap({
  data,
  width = 1200,
  height = 600,
  onTileClick,
}: WatchListHeatmapProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const containerRectRef = useRef<DOMRect | null>(null);
  const router = useRouter();
  const { resolvedTheme } = useTheme();

  const isDark = resolvedTheme === 'dark';
  const strokeColor = isDark ? themeColors.dark.border : themeColors.light.border;
  const neutralColor = isDark ? themeColors.dark.bgSecondary : themeColors.light.bgSecondary;

  const useSectorGrouping = data.tiles.length >= SECTOR_GROUPING_THRESHOLD;
  const sectorHeaderH = 22;

  useEffect(() => {
    if (!svgRef.current || !data.tiles.length) return;

    d3.select(svgRef.current).selectAll('*').remove();
    const svg = d3.select(svgRef.current);

    // Issue 2: symmetric color scale
    const colorScale = getSymmetricColorScale(data.color_scheme, neutralColor);

    // Build hierarchy — Issue 3: use raw size_value (no artificial floor)
    // Issue 5: two-level hierarchy when sector grouping is enabled
    let root: d3.HierarchyNode<any>;

    if (useSectorGrouping) {
      const sectorMap = new Map<string, HeatmapTile[]>();
      data.tiles.forEach((tile) => {
        const sector = tile.sector || 'Other';
        if (!sectorMap.has(sector)) sectorMap.set(sector, []);
        sectorMap.get(sector)!.push(tile);
      });

      root = d3
        .hierarchy({
          name: 'root',
          children: Array.from(sectorMap.entries()).map(([name, children]) => ({
            name,
            children,
          })),
        })
        .sum((d: any) => (d.size_value != null && !d.children ? Math.max(d.size_value, 1) : 0))
        .sort((a, b) => (b.value || 0) - (a.value || 0));
    } else {
      root = d3
        .hierarchy({ children: data.tiles } as any)
        .sum((d: any) => Math.max(d.size_value || 1, 1))
        .sort((a, b) => (b.value || 0) - (a.value || 0));
    }

    const treemap = d3
      .treemap<any>()
      .size([width, height])
      .paddingInner(2)
      .paddingOuter(2)
      .paddingTop((d: any) => (useSectorGrouping && d.depth === 1 ? sectorHeaderH : 2))
      .round(true);

    treemap(root);

    // Issue 5: draw sector backgrounds and labels
    if (useSectorGrouping && root.children) {
      const sectorBg = isDark
        ? ['rgba(255,255,255,0.03)', 'rgba(255,255,255,0.05)']
        : ['rgba(0,0,0,0.02)', 'rgba(0,0,0,0.04)'];

      root.children.forEach((sectorNode: any, i: number) => {
        const g = svg.append('g');

        g.append('rect')
          .attr('x', sectorNode.x0)
          .attr('y', sectorNode.y0)
          .attr('width', sectorNode.x1 - sectorNode.x0)
          .attr('height', sectorNode.y1 - sectorNode.y0)
          .attr('fill', sectorBg[i % 2])
          .attr('stroke', isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)')
          .attr('stroke-width', 1)
          .attr('rx', 4);

        const sW = sectorNode.x1 - sectorNode.x0;
        if (sW > 60) {
          g.append('text')
            .attr('x', sectorNode.x0 + 6)
            .attr('y', sectorNode.y0 + 15)
            .text(sectorNode.data.name)
            .style('font-size', '11px')
            .style('font-weight', '600')
            .style('fill', isDark ? 'rgba(255,255,255,0.5)' : 'rgba(0,0,0,0.45)')
            .style('text-transform', 'uppercase')
            .style('letter-spacing', '0.5px')
            .style('pointer-events', 'none');
        }
      });
    }

    // Draw leaf tiles
    const leaves = root.leaves();
    const nodes = svg
      .selectAll('g.tile')
      .data(leaves)
      .join('g')
      .attr('class', 'tile')
      .attr('transform', (d: any) => `translate(${d.x0},${d.y0})`);

    // Tile rectangles — Issue 7: hover affordance
    nodes
      .append('rect')
      .attr('class', 'tile-rect')
      .attr('width', (d: any) => Math.max(d.x1 - d.x0, 0))
      .attr('height', (d: any) => Math.max(d.y1 - d.y0, 0))
      .attr('fill', (d: any) => colorScale(d.data.color_value))
      .attr('stroke', strokeColor)
      .attr('stroke-width', 1)
      .attr('rx', 3)
      .style('cursor', 'pointer')
      .on('mouseover', function (event: any, d: any) {
        d3.select(this)
          .attr('stroke', '#ffffff')
          .attr('stroke-width', 2)
          .style('filter', 'brightness(1.05)');
        showTooltip(event, d.data);
      })
      .on('mousemove', function (event: any) {
        moveTooltip(event);
      })
      .on('mouseout', function () {
        d3.select(this).attr('stroke', strokeColor).attr('stroke-width', 1).style('filter', null);
        hideTooltip();
      })
      .on('click', (_event: any, d: any) => {
        if (onTileClick) onTileClick(d.data.symbol);
        else router.push(`/ticker/${d.data.symbol}`);
      });

    // Issue 6: dynamic label placement
    nodes.each(function (d: any) {
      const g = d3.select(this);
      const tileW = d.x1 - d.x0;
      const tileH = d.y1 - d.y0;
      const minDim = Math.min(tileW, tileH);

      if (minDim < 30) return; // too small for any label

      const bgColor = colorScale(d.data.color_value);
      const textColor = getTextColor(bgColor);
      const tileArea = tileW * tileH;

      // Font sizing scaled by tile area (Issue 6)
      const tickerSize = Math.max(11, Math.min(20, Math.sqrt(tileArea) / 8));
      const changeSize = Math.max(9, Math.min(14, tickerSize * 0.7));

      // Vertical positioning: bottom-weighted for tall tiles (Issue 6)
      const isLarge = tileW > 200 && tileH > 80;
      const isTall = tileH > tileW * 1.5;

      let baseY: number;
      if (isTall) baseY = tileH * 0.55;
      else if (isLarge) baseY = tileH * 0.4;
      else baseY = tileH * 0.45;

      // Ticker symbol
      g.append('text')
        .attr('x', tileW / 2)
        .attr('y', baseY)
        .attr('text-anchor', 'middle')
        .attr('dominant-baseline', 'central')
        .attr('fill', textColor)
        .style('font-weight', 'bold')
        .style('font-size', `${tickerSize}px`)
        .style('pointer-events', 'none')
        .text(d.data.symbol);

      // Change percentage
      if (minDim >= 40) {
        g.append('text')
          .attr('x', tileW / 2)
          .attr('y', baseY + tickerSize + 2)
          .attr('text-anchor', 'middle')
          .attr('dominant-baseline', 'central')
          .attr('fill', textColor)
          .style('font-size', `${changeSize}px`)
          .style('opacity', '0.9')
          .style('pointer-events', 'none')
          .text(d.data.color_label);
      }

      // Company name + price on large tiles (Issue 6)
      if (isLarge) {
        const nameSize = Math.max(9, Math.min(12, tickerSize * 0.6));
        const maxChars = Math.floor(tileW / (nameSize * 0.55));
        const truncName =
          d.data.name.length > maxChars
            ? d.data.name.slice(0, maxChars - 1) + '\u2026'
            : d.data.name;

        g.append('text')
          .attr('x', tileW / 2)
          .attr('y', baseY - tickerSize - 4)
          .attr('text-anchor', 'middle')
          .attr('dominant-baseline', 'central')
          .attr('fill', textColor)
          .style('font-size', `${nameSize}px`)
          .style('opacity', '0.7')
          .style('pointer-events', 'none')
          .text(truncName);

        if (tileH > 100 && d.data.current_price != null) {
          g.append('text')
            .attr('x', tileW / 2)
            .attr('y', baseY + tickerSize + changeSize + 6)
            .attr('text-anchor', 'middle')
            .attr('dominant-baseline', 'central')
            .attr('fill', textColor)
            .style('font-size', `${nameSize}px`)
            .style('opacity', '0.6')
            .style('pointer-events', 'none')
            .text(`$${d.data.current_price.toFixed(2)}`);
        }
      }
    });

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    data,
    width,
    height,
    router,
    onTileClick,
    resolvedTheme,
    strokeColor,
    neutralColor,
    useSectorGrouping,
    isDark,
  ]);

  // ─── Tooltip helpers (Issue 7: container-relative positioning) ──

  const showTooltip = (event: MouseEvent, tile: HeatmapTile) => {
    const tooltip = tooltipRef.current;
    if (!tooltip) return;

    // Cache container rect once on tooltip show to avoid layout reads on every mousemove
    if (containerRef.current) {
      containerRectRef.current = containerRef.current.getBoundingClientRect();
    }

    tooltip.textContent = '';

    tooltip.appendChild(el('div', 'font-bold text-lg mb-2', `${tile.symbol} - ${tile.name}`));

    const grid = el('div', 'grid grid-cols-2 gap-2 text-sm');

    const addRow = (label: string, value: string, valueClass = 'font-medium') => {
      grid.appendChild(el('div', 'text-ic-text-muted', label));
      grid.appendChild(el('div', valueClass, value));
    };

    const addDivider = () => {
      grid.appendChild(el('div', 'col-span-2 border-t border-ic-border mt-1 pt-1'));
    };

    addRow('Price:', `$${tile.current_price.toFixed(2)}`);

    const sign = tile.price_change >= 0 ? '+' : '';
    const chgClass =
      tile.price_change >= 0 ? 'font-medium text-ic-positive' : 'font-medium text-ic-negative';
    addRow(
      'Change:',
      `${sign}${tile.price_change.toFixed(2)} (${tile.price_change_pct.toFixed(2)}%)`,
      chgClass
    );

    if (tile.market_cap) addRow('Market Cap:', tile.size_label);
    if (tile.volume) addRow('Volume:', formatVolume(tile.volume));

    if (tile.reddit_rank) {
      addDivider();
      addRow('Reddit Rank:', `#${tile.reddit_rank}`, 'font-medium text-purple-400');
    }
    if (tile.reddit_mentions) addRow('Reddit Mentions:', tile.reddit_mentions.toLocaleString());
    if (tile.reddit_popularity) addRow('Reddit Score:', `${tile.reddit_popularity.toFixed(1)}/100`);
    if (tile.reddit_trend) {
      const arrow =
        tile.reddit_trend === 'rising'
          ? '\u2191'
          : tile.reddit_trend === 'falling'
            ? '\u2193'
            : '\u2192';
      const tColor =
        tile.reddit_trend === 'rising'
          ? 'font-medium text-ic-positive'
          : tile.reddit_trend === 'falling'
            ? 'font-medium text-ic-negative'
            : 'font-medium text-ic-text-muted';
      const rc = tile.reddit_rank_change
        ? ` (${tile.reddit_rank_change > 0 ? '+' : ''}${tile.reddit_rank_change})`
        : '';
      addRow('Reddit Trend:', `${arrow} ${tile.reddit_trend}${rc}`, tColor);
    }

    if (tile.target_buy_price) {
      addDivider();
      addRow('Target Buy:', `$${tile.target_buy_price.toFixed(2)}`, 'font-medium text-ic-blue');
    }
    if (tile.target_sell_price)
      addRow(
        'Target Sell:',
        `$${tile.target_sell_price.toFixed(2)}`,
        'font-medium text-orange-400'
      );

    tooltip.appendChild(grid);

    if (tile.notes) {
      tooltip.appendChild(el('div', 'mt-2 text-sm text-ic-text-muted italic', tile.notes));
    }

    if (tile.tags && tile.tags.length > 0) {
      const tagContainer = el('div', 'mt-2 flex flex-wrap gap-1');
      tile.tags.forEach((tag) => {
        tagContainer.appendChild(el('span', 'text-xs bg-ic-bg-secondary px-2 py-1 rounded', tag));
      });
      tooltip.appendChild(tagContainer);
    }

    tooltip.style.display = 'block';
    moveTooltip(event);
  };

  const moveTooltip = (event: MouseEvent) => {
    const tooltip = tooltipRef.current;
    const cr = containerRectRef.current;
    if (!tooltip || !cr) return;

    const tw = tooltip.offsetWidth;
    const th = tooltip.offsetHeight;

    let x = event.clientX - cr.left + 12;
    let y = event.clientY - cr.top + 12;

    // Clamp within container bounds
    if (x + tw > cr.width) x = event.clientX - cr.left - tw - 12;
    if (y + th > cr.height) y = event.clientY - cr.top - th - 12;
    x = Math.max(0, x);
    y = Math.max(0, y);

    tooltip.style.left = `${x}px`;
    tooltip.style.top = `${y}px`;
  };

  const hideTooltip = () => {
    const tooltip = tooltipRef.current;
    if (tooltip) tooltip.style.display = 'none';
    containerRectRef.current = null;
  };

  return (
    <div ref={containerRef} className="relative">
      <svg ref={svgRef} width={width} height={height} className="bg-ic-bg-secondary rounded-lg" />
      <ColorLegend scheme={data.color_scheme} neutralColor={neutralColor} />
      <div
        ref={tooltipRef}
        className="absolute hidden bg-ic-surface p-4 rounded-lg border border-ic-border max-w-sm z-50 shadow-lg"
        style={{ pointerEvents: 'none' }}
      />
    </div>
  );
}
