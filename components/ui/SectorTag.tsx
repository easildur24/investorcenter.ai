'use client';

import { getSectorConfig } from '@/config/gics-sectors';

interface SectorTagProps {
  /** Sector name from the API (e.g., "Technology", "Healthcare") */
  sector: string;
  /** Size variant */
  size?: 'sm' | 'md';
  /** Additional className */
  className?: string;
}

/**
 * GICS sector pill badge — color-coded by sector.
 * Falls back to a neutral style for unknown sectors.
 */
export default function SectorTag({ sector, size = 'sm', className = '' }: SectorTagProps) {
  const config = getSectorConfig(sector);

  const bgColor = config?.color || '#6B7280';
  const sizeClasses = size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-1';

  return (
    <span
      className={`inline-flex items-center rounded-full font-medium leading-none whitespace-nowrap ${sizeClasses} ${className}`}
      style={{
        backgroundColor: `${bgColor}20`,
        color: bgColor,
      }}
    >
      {config?.label || sector}
    </span>
  );
}
