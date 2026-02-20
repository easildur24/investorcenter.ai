'use client';

import { useCallback, useEffect, useRef } from 'react';
import * as d3 from 'd3';
import { HeatmapTile, HeatmapData } from '@/lib/api/heatmap';
import { useRouter } from 'next/navigation';
import { useTheme } from '@/lib/contexts/ThemeContext';
import { themeColors } from '@/lib/theme';

interface WatchListHeatmapProps {
  data: HeatmapData;
  width?: number;
  height?: number;
  onTileClick?: (symbol: string) => void;
}

export default function WatchListHeatmap({
  data,
  width = 1200,
  height = 600,
  onTileClick,
}: WatchListHeatmapProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const { resolvedTheme } = useTheme();

  // Theme-aware colors
  const isDark = resolvedTheme === 'dark';
  const strokeColor = isDark ? themeColors.dark.border : themeColors.light.border;
  const strokeHighlight = isDark ? '#ffffff' : '#000000';
  const neutralColor = isDark ? themeColors.dark.bgSecondary : themeColors.light.bgSecondary;

  useEffect(() => {
    if (!svgRef.current || !data.tiles.length) return;

    // Clear previous render
    d3.select(svgRef.current).selectAll('*').remove();

    // Create treemap layout with proper sizing
    const root = d3
      .hierarchy({ children: data.tiles } as any)
      .sum((d: any) => {
        // Ensure we have a valid size_value, use a minimum of 10 for visibility
        const sizeValue = d.size_value || 10;
        // For better visual differentiation, we can apply a slight scaling
        return Math.max(sizeValue, 10);
      })
      .sort((a, b) => (b.value || 0) - (a.value || 0));

    const treemap = d3
      .treemap<any>()
      .size([width, height])
      .padding(3) // Increased padding for better visual separation
      .paddingOuter(4) // Extra padding on edges
      .round(true);

    treemap(root);

    // Color scale based on color metric (theme-aware)
    const colorScale = getColorScale(
      data.color_scheme,
      data.min_color_value,
      data.max_color_value,
      neutralColor
    );

    const svg = d3.select(svgRef.current);

    // Create groups for each tile
    const nodes = svg
      .selectAll('g')
      .data(root.leaves())
      .join('g')
      .attr('transform', (d: any) => `translate(${d.x0},${d.y0})`);

    // Add rectangles
    nodes
      .append('rect')
      .attr('width', (d: any) => d.x1 - d.x0)
      .attr('height', (d: any) => d.y1 - d.y0)
      .attr('fill', (d: any) => colorScale(d.data.color_value))
      .attr('stroke', strokeColor)
      .attr('stroke-width', 2)
      .attr('rx', 4)
      .style('cursor', 'pointer')
      .on('mouseover', function (event: any, d: any) {
        // Highlight tile
        d3.select(this).attr('stroke', strokeHighlight).attr('stroke-width', 3);

        // Show tooltip
        showTooltip(event, d.data);
      })
      .on('mouseout', function () {
        // Remove highlight
        d3.select(this).attr('stroke', strokeColor).attr('stroke-width', 2);

        // Hide tooltip
        hideTooltip();
      })
      .on('click', (event: any, d: any) => {
        if (onTileClick) {
          onTileClick(d.data.symbol);
        } else {
          router.push(`/ticker/${d.data.symbol}`);
        }
      });

    // Add symbol text
    nodes
      .append('text')
      .attr('x', (d: any) => (d.x1 - d.x0) / 2)
      .attr('y', (d: any) => (d.y1 - d.y0) / 2 - 8)
      .attr('text-anchor', 'middle')
      .attr('fill', (d: any) => getTextColor(colorScale(d.data.color_value)))
      .style('font-weight', 'bold')
      .style('font-size', (d: any) => {
        const tileSize = Math.min(d.x1 - d.x0, d.y1 - d.y0);
        return `${Math.max(10, Math.min(16, tileSize / 8))}px`;
      })
      .style('pointer-events', 'none')
      .text((d: any) => d.data.symbol);

    // Add change percentage text
    if (data.color_metric === 'price_change_pct') {
      nodes
        .append('text')
        .attr('x', (d: any) => (d.x1 - d.x0) / 2)
        .attr('y', (d: any) => (d.y1 - d.y0) / 2 + 12)
        .attr('text-anchor', 'middle')
        .attr('fill', (d: any) => getTextColor(colorScale(d.data.color_value)))
        .style('font-size', (d: any) => {
          const tileSize = Math.min(d.x1 - d.x0, d.y1 - d.y0);
          return `${Math.max(8, Math.min(12, tileSize / 12))}px`;
        })
        .style('pointer-events', 'none')
        .text((d: any) => d.data.color_label);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    data,
    width,
    height,
    router,
    onTileClick,
    resolvedTheme,
    strokeColor,
    strokeHighlight,
    neutralColor,
  ]);

  // Helper to safely create a DOM element with text content (prevents XSS)
  // Defined before showTooltip so it can be used inside useCallback
  const el = (
    tag: string,
    className: string,
    text?: string | number,
    children?: HTMLElement[]
  ): HTMLElement => {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text != null) node.textContent = String(text);
    if (children) children.forEach((c) => node.appendChild(c));
    return node;
  };

  const showTooltip = (event: any, tile: HeatmapTile) => {
    const tooltip = tooltipRef.current;
    if (!tooltip) return;

    // Clear previous content safely
    tooltip.textContent = '';

    // Header
    tooltip.appendChild(el('div', 'font-bold text-lg mb-2', `${tile.symbol} - ${tile.name}`));

    // Grid of key-value pairs
    const grid = el('div', 'grid grid-cols-2 gap-2 text-sm');

    const addRow = (label: string, value: string, valueClass = 'font-medium') => {
      grid.appendChild(el('div', 'text-ic-text-muted', label));
      grid.appendChild(el('div', valueClass, value));
    };

    const addDivider = () => {
      grid.appendChild(el('div', 'col-span-2 border-t border-ic-border mt-1 pt-1'));
    };

    addRow('Price:', `$${tile.current_price.toFixed(2)}`);

    const changeSign = tile.price_change >= 0 ? '+' : '';
    const changeColor =
      tile.price_change >= 0 ? 'font-medium text-ic-positive' : 'font-medium text-ic-negative';
    addRow(
      'Change:',
      `${changeSign}${tile.price_change.toFixed(2)} (${tile.price_change_pct.toFixed(2)}%)`,
      changeColor
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
      const trendArrow =
        tile.reddit_trend === 'rising' ? '↑' : tile.reddit_trend === 'falling' ? '↓' : '→';
      const trendColor =
        tile.reddit_trend === 'rising'
          ? 'font-medium text-ic-positive'
          : tile.reddit_trend === 'falling'
            ? 'font-medium text-ic-negative'
            : 'font-medium text-ic-text-muted';
      const rankChange = tile.reddit_rank_change
        ? ` (${tile.reddit_rank_change > 0 ? '+' : ''}${tile.reddit_rank_change})`
        : '';
      addRow('Reddit Trend:', `${trendArrow} ${tile.reddit_trend}${rankChange}`, trendColor);
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

    // Notes (safely escaped)
    if (tile.notes) {
      tooltip.appendChild(el('div', 'mt-2 text-sm text-ic-text-muted italic', tile.notes));
    }

    // Tags (safely escaped)
    if (tile.tags.length > 0) {
      const tagContainer = el('div', 'mt-2 flex flex-wrap gap-1');
      tile.tags.forEach((tag) => {
        tagContainer.appendChild(el('span', 'text-xs bg-ic-bg-secondary px-2 py-1 rounded', tag));
      });
      tooltip.appendChild(tagContainer);
    }

    tooltip.style.display = 'block';
    tooltip.style.left = `${event.pageX + 10}px`;
    tooltip.style.top = `${event.pageY + 10}px`;
  };

  const hideTooltip = () => {
    const tooltip = tooltipRef.current;
    if (tooltip) {
      tooltip.style.display = 'none';
    }
  };

  const formatVolume = (vol: number) => {
    if (vol >= 1e9) return `${(vol / 1e9).toFixed(1)}B`;
    if (vol >= 1e6) return `${(vol / 1e6).toFixed(1)}M`;
    if (vol >= 1e3) return `${(vol / 1e3).toFixed(1)}K`;
    return vol.toString();
  };

  return (
    <div className="relative">
      <svg ref={svgRef} width={width} height={height} className="bg-ic-bg-secondary rounded-lg" />
      <div
        ref={tooltipRef}
        className="absolute hidden bg-ic-surface p-4 rounded-lg border border-ic-border max-w-sm z-50"
        style={{ pointerEvents: 'none' }}
      />
    </div>
  );
}

// Helper functions

function getColorScale(scheme: string, min: number, max: number, neutralColor: string) {
  // Use theme accent colors for positive/negative
  const positiveColor = themeColors.accent.positive;
  const negativeColor = themeColors.accent.negative;
  const blueColor = themeColors.accent.blue;

  switch (scheme) {
    case 'red_green':
      return d3
        .scaleLinear<string>()
        .domain([min, 0, max])
        .range([negativeColor, neutralColor, positiveColor]);

    case 'blue_red':
      return d3
        .scaleLinear<string>()
        .domain([min, 0, max])
        .range([blueColor, neutralColor, negativeColor]);

    case 'heatmap':
      return d3.scaleSequential(d3.interpolateRdYlGn).domain([min, max]);

    default:
      return d3
        .scaleLinear<string>()
        .domain([min, 0, max])
        .range([negativeColor, neutralColor, positiveColor]);
  }
}

function getTextColor(bgColor: string): string {
  // Convert hex/rgb to luminance
  const rgb = d3.rgb(bgColor);
  const luminance = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255;
  return luminance > 0.5 ? '#000000' : '#FFFFFF';
}
