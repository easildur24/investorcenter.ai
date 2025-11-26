'use client';

import { InformationCircleIcon } from '@heroicons/react/24/outline';

export type DataSourceType = 'realtime' | 'sec-filing' | 'calculated' | 'estimated';

interface DataSourceBadgeProps {
  type: DataSourceType;
  className?: string;
  showTooltip?: boolean;
}

const sourceConfig: Record<DataSourceType, {
  label: string;
  description: string;
  bgColor: string;
  textColor: string;
  borderColor: string;
}> = {
  'realtime': {
    label: 'Real-time',
    description: 'Live market data updated in real-time from Polygon.io',
    bgColor: 'bg-green-50',
    textColor: 'text-green-700',
    borderColor: 'border-green-200',
  },
  'sec-filing': {
    label: 'SEC Filing',
    description: 'Data sourced from official SEC filings (10-K, 10-Q)',
    bgColor: 'bg-blue-50',
    textColor: 'text-blue-700',
    borderColor: 'border-blue-200',
  },
  'calculated': {
    label: 'Calculated',
    description: 'Derived from multiple data sources using proprietary algorithms',
    bgColor: 'bg-purple-50',
    textColor: 'text-purple-700',
    borderColor: 'border-purple-200',
  },
  'estimated': {
    label: 'Estimated',
    description: 'Estimated value - may not reflect actual current data',
    bgColor: 'bg-amber-50',
    textColor: 'text-amber-700',
    borderColor: 'border-amber-200',
  },
};

export default function DataSourceBadge({
  type,
  className = '',
  showTooltip = true,
}: DataSourceBadgeProps) {
  const config = sourceConfig[type];

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full border ${config.bgColor} ${config.textColor} ${config.borderColor} ${className}`}
      title={showTooltip ? config.description : undefined}
    >
      {type === 'estimated' && (
        <InformationCircleIcon className="w-3 h-3" />
      )}
      {config.label}
    </span>
  );
}

// Inline badge for use within metric rows
export function DataSourceIndicator({
  type,
  className = '',
}: {
  type: DataSourceType;
  className?: string;
}) {
  const config = sourceConfig[type];

  return (
    <span
      className={`inline-block w-2 h-2 rounded-full ${config.bgColor.replace('50', '400')} ${className}`}
      title={config.description}
    />
  );
}

// Multiple source badge for combined data
export function MultiSourceBadge({
  sources,
  className = '',
}: {
  sources: DataSourceType[];
  className?: string;
}) {
  if (sources.length === 0) return null;

  // Show primary source
  const primarySource = sources[0];
  const config = sourceConfig[primarySource];

  return (
    <span
      className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded-full border ${config.bgColor} ${config.textColor} ${config.borderColor} ${className}`}
      title={`Sources: ${sources.map(s => sourceConfig[s].label).join(', ')}`}
    >
      {config.label}
      {sources.length > 1 && (
        <span className="text-gray-400">+{sources.length - 1}</span>
      )}
    </span>
  );
}
