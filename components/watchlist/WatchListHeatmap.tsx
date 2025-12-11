'use client';

import { useEffect, useRef } from 'react';
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
    const root = d3.hierarchy({ children: data.tiles } as any)
      .sum((d: any) => {
        // Ensure we have a valid size_value, use a minimum of 10 for visibility
        const sizeValue = d.size_value || 10;
        // For better visual differentiation, we can apply a slight scaling
        return Math.max(sizeValue, 10);
      })
      .sort((a, b) => (b.value || 0) - (a.value || 0));

    const treemap = d3.treemap<any>()
      .size([width, height])
      .padding(3)  // Increased padding for better visual separation
      .paddingOuter(4)  // Extra padding on edges
      .round(true);

    treemap(root);

    // Color scale based on color metric (theme-aware)
    const colorScale = getColorScale(data.color_scheme, data.min_color_value, data.max_color_value, neutralColor);

    const svg = d3.select(svgRef.current);

    // Create groups for each tile
    const nodes = svg.selectAll('g')
      .data(root.leaves())
      .join('g')
      .attr('transform', (d: any) => `translate(${d.x0},${d.y0})`);

    // Add rectangles
    nodes.append('rect')
      .attr('width', (d: any) => d.x1 - d.x0)
      .attr('height', (d: any) => d.y1 - d.y0)
      .attr('fill', (d: any) => colorScale(d.data.color_value))
      .attr('stroke', strokeColor)
      .attr('stroke-width', 2)
      .attr('rx', 4)
      .style('cursor', 'pointer')
      .on('mouseover', function(event: any, d: any) {
        // Highlight tile
        d3.select(this)
          .attr('stroke', strokeHighlight)
          .attr('stroke-width', 3);

        // Show tooltip
        showTooltip(event, d.data);
      })
      .on('mouseout', function() {
        // Remove highlight
        d3.select(this)
          .attr('stroke', strokeColor)
          .attr('stroke-width', 2);

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
    nodes.append('text')
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
      nodes.append('text')
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

  }, [data, width, height, router, onTileClick, resolvedTheme, strokeColor, strokeHighlight, neutralColor]);

  const showTooltip = (event: any, tile: HeatmapTile) => {
    const tooltip = tooltipRef.current;
    if (!tooltip) return;

    tooltip.innerHTML = `
      <div class="font-bold text-lg mb-2">${tile.symbol} - ${tile.name}</div>
      <div class="grid grid-cols-2 gap-2 text-sm">
        <div class="text-ic-text-muted">Price:</div>
        <div class="font-medium">$${tile.current_price.toFixed(2)}</div>

        <div class="text-ic-text-muted">Change:</div>
        <div class="font-medium ${tile.price_change >= 0 ? 'text-ic-positive' : 'text-ic-negative'}">
          ${tile.price_change >= 0 ? '+' : ''}${tile.price_change.toFixed(2)} (${tile.price_change_pct.toFixed(2)}%)
        </div>

        ${tile.market_cap ? `
          <div class="text-ic-text-muted">Market Cap:</div>
          <div class="font-medium">${tile.size_label}</div>
        ` : ''}

        ${tile.volume ? `
          <div class="text-ic-text-muted">Volume:</div>
          <div class="font-medium">${formatVolume(tile.volume)}</div>
        ` : ''}

        ${tile.reddit_rank ? `
          <div class="col-span-2 border-t border-ic-border mt-1 pt-1"></div>
          <div class="text-ic-text-muted">Reddit Rank:</div>
          <div class="font-medium text-purple-600">#${tile.reddit_rank}</div>
        ` : ''}

        ${tile.reddit_mentions ? `
          <div class="text-ic-text-muted">Reddit Mentions:</div>
          <div class="font-medium">${tile.reddit_mentions.toLocaleString()}</div>
        ` : ''}

        ${tile.reddit_popularity ? `
          <div class="text-ic-text-muted">Reddit Score:</div>
          <div class="font-medium">${tile.reddit_popularity.toFixed(1)}/100</div>
        ` : ''}

        ${tile.reddit_trend ? `
          <div class="text-ic-text-muted">Reddit Trend:</div>
          <div class="font-medium ${
            tile.reddit_trend === 'rising' ? 'text-ic-positive' :
            tile.reddit_trend === 'falling' ? 'text-ic-negative' :
            'text-ic-text-muted'
          }">
            ${tile.reddit_trend === 'rising' ? '↑' : tile.reddit_trend === 'falling' ? '↓' : '→'} ${tile.reddit_trend}
            ${tile.reddit_rank_change ? ` (${tile.reddit_rank_change > 0 ? '+' : ''}${tile.reddit_rank_change})` : ''}
          </div>
        ` : ''}

        ${tile.target_buy_price ? `
          <div class="col-span-2 border-t border-ic-border mt-1 pt-1"></div>
          <div class="text-ic-text-muted">Target Buy:</div>
          <div class="font-medium text-ic-blue">$${tile.target_buy_price.toFixed(2)}</div>
        ` : ''}

        ${tile.target_sell_price ? `
          <div class="text-ic-text-muted">Target Sell:</div>
          <div class="font-medium text-orange-600">$${tile.target_sell_price.toFixed(2)}</div>
        ` : ''}
      </div>
      ${tile.notes ? `<div class="mt-2 text-sm text-ic-text-muted italic">${tile.notes}</div>` : ''}
      ${tile.tags.length > 0 ? `
        <div class="mt-2 flex flex-wrap gap-1">
          ${tile.tags.map(tag => `<span class="text-xs bg-ic-bg-secondary px-2 py-1 rounded">${tag}</span>`).join('')}
        </div>
      ` : ''}
    `;

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
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range([negativeColor, neutralColor, positiveColor]);

    case 'blue_red':
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range([blueColor, neutralColor, negativeColor]);

    case 'heatmap':
      return d3.scaleSequential(d3.interpolateRdYlGn)
        .domain([min, max]);

    default:
      return d3.scaleLinear<string>()
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
