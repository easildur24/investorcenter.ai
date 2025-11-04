'use client';

import { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import { HeatmapTile, HeatmapData } from '@/lib/api/heatmap';
import { useRouter } from 'next/navigation';

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

    // Color scale based on color metric
    const colorScale = getColorScale(data.color_scheme, data.min_color_value, data.max_color_value);

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
      .attr('stroke', '#fff')
      .attr('stroke-width', 2)
      .attr('rx', 4)
      .style('cursor', 'pointer')
      .on('mouseover', function(event: any, d: any) {
        // Highlight tile
        d3.select(this)
          .attr('stroke', '#000')
          .attr('stroke-width', 3);

        // Show tooltip
        showTooltip(event, d.data);
      })
      .on('mouseout', function() {
        // Remove highlight
        d3.select(this)
          .attr('stroke', '#fff')
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

  }, [data, width, height, router, onTileClick]);

  const showTooltip = (event: any, tile: HeatmapTile) => {
    const tooltip = tooltipRef.current;
    if (!tooltip) return;

    tooltip.innerHTML = `
      <div class="font-bold text-lg mb-2">${tile.symbol} - ${tile.name}</div>
      <div class="grid grid-cols-2 gap-2 text-sm">
        <div class="text-gray-600">Price:</div>
        <div class="font-medium">$${tile.current_price.toFixed(2)}</div>

        <div class="text-gray-600">Change:</div>
        <div class="font-medium ${tile.price_change >= 0 ? 'text-green-600' : 'text-red-600'}">
          ${tile.price_change >= 0 ? '+' : ''}${tile.price_change.toFixed(2)} (${tile.price_change_pct.toFixed(2)}%)
        </div>

        ${tile.market_cap ? `
          <div class="text-gray-600">Market Cap:</div>
          <div class="font-medium">${tile.size_label}</div>
        ` : ''}

        ${tile.volume ? `
          <div class="text-gray-600">Volume:</div>
          <div class="font-medium">${formatVolume(tile.volume)}</div>
        ` : ''}

        ${tile.reddit_rank ? `
          <div class="col-span-2 border-t border-gray-200 mt-1 pt-1"></div>
          <div class="text-gray-600">Reddit Rank:</div>
          <div class="font-medium text-purple-600">#${tile.reddit_rank}</div>
        ` : ''}

        ${tile.reddit_mentions ? `
          <div class="text-gray-600">Reddit Mentions:</div>
          <div class="font-medium">${tile.reddit_mentions.toLocaleString()}</div>
        ` : ''}

        ${tile.reddit_popularity ? `
          <div class="text-gray-600">Reddit Score:</div>
          <div class="font-medium">${tile.reddit_popularity.toFixed(1)}/100</div>
        ` : ''}

        ${tile.reddit_trend ? `
          <div class="text-gray-600">Reddit Trend:</div>
          <div class="font-medium ${
            tile.reddit_trend === 'rising' ? 'text-green-600' :
            tile.reddit_trend === 'falling' ? 'text-red-600' :
            'text-gray-600'
          }">
            ${tile.reddit_trend === 'rising' ? '↑' : tile.reddit_trend === 'falling' ? '↓' : '→'} ${tile.reddit_trend}
            ${tile.reddit_rank_change ? ` (${tile.reddit_rank_change > 0 ? '+' : ''}${tile.reddit_rank_change})` : ''}
          </div>
        ` : ''}

        ${tile.target_buy_price ? `
          <div class="col-span-2 border-t border-gray-200 mt-1 pt-1"></div>
          <div class="text-gray-600">Target Buy:</div>
          <div class="font-medium text-blue-600">$${tile.target_buy_price.toFixed(2)}</div>
        ` : ''}

        ${tile.target_sell_price ? `
          <div class="text-gray-600">Target Sell:</div>
          <div class="font-medium text-orange-600">$${tile.target_sell_price.toFixed(2)}</div>
        ` : ''}
      </div>
      ${tile.notes ? `<div class="mt-2 text-sm text-gray-600 italic">${tile.notes}</div>` : ''}
      ${tile.tags.length > 0 ? `
        <div class="mt-2 flex flex-wrap gap-1">
          ${tile.tags.map(tag => `<span class="text-xs bg-gray-200 px-2 py-1 rounded">${tag}</span>`).join('')}
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
      <svg ref={svgRef} width={width} height={height} className="bg-gray-50 rounded-lg" />
      <div
        ref={tooltipRef}
        className="absolute hidden bg-white p-4 rounded-lg shadow-lg border border-gray-200 max-w-sm z-50"
        style={{ pointerEvents: 'none' }}
      />
    </div>
  );
}

// Helper functions

function getColorScale(scheme: string, min: number, max: number) {
  switch (scheme) {
    case 'red_green':
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range(['#EF4444', '#F3F4F6', '#10B981']);

    case 'blue_red':
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range(['#3B82F6', '#F3F4F6', '#EF4444']);

    case 'heatmap':
      return d3.scaleSequential(d3.interpolateRdYlGn)
        .domain([min, max]);

    default:
      return d3.scaleLinear<string>()
        .domain([min, 0, max])
        .range(['#EF4444', '#F3F4F6', '#10B981']);
  }
}

function getTextColor(bgColor: string): string {
  // Convert hex/rgb to luminance
  const rgb = d3.rgb(bgColor);
  const luminance = (0.299 * rgb.r + 0.587 * rgb.g + 0.114 * rgb.b) / 255;
  return luminance > 0.5 ? '#000000' : '#FFFFFF';
}
