'use client';

import { cn } from '@/lib/utils';
import { Tooltip } from '@/components/ui/Tooltip';
import type { LifecycleStage } from '@/lib/types/fundamentals';

interface LifecycleBadgeProps {
  stage: LifecycleStage;
  description: string;
  size?: 'sm' | 'md';
}

const stageColors: Record<LifecycleStage, { bg: string; text: string }> = {
  hypergrowth: { bg: 'bg-purple-500/15', text: 'text-purple-400' },
  growth: { bg: 'bg-blue-500/15', text: 'text-blue-400' },
  mature: { bg: 'bg-slate-500/15', text: 'text-slate-300' },
  value: { bg: 'bg-amber-500/15', text: 'text-amber-400' },
  turnaround: { bg: 'bg-orange-500/15', text: 'text-orange-400' },
};

const stageLabels: Record<LifecycleStage, string> = {
  hypergrowth: 'Hypergrowth',
  growth: 'Growth',
  mature: 'Mature',
  value: 'Value',
  turnaround: 'Turnaround',
};

const sizeClasses: Record<string, { wrapper: string; text: string }> = {
  sm: { wrapper: 'px-2 py-1', text: 'text-xs' },
  md: { wrapper: 'px-3 py-1.5', text: 'text-sm' },
};

export function LifecycleBadge({ stage, description, size = 'md' }: LifecycleBadgeProps) {
  const colors = stageColors[stage];
  const sizes = sizeClasses[size];
  const label = stageLabels[stage];

  return (
    <Tooltip
      content={
        <div className="max-w-xs">
          <div className="font-medium text-ic-text-primary mb-1">{label} Stage</div>
          <div className="text-xs text-ic-text-muted">{description}</div>
        </div>
      }
      position="bottom"
    >
      <span
        className={cn(
          'inline-flex items-center rounded-full font-medium cursor-help',
          colors.bg,
          colors.text,
          sizes.wrapper,
          sizes.text
        )}
      >
        {label}
      </span>
    </Tooltip>
  );
}
