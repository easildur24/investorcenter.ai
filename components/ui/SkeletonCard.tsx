'use client';

/**
 * Reusable skeleton primitives for loading states.
 * Matches the existing `animate-pulse` + `bg-ic-bg-tertiary` pattern.
 */

interface SkeletonProps {
  className?: string;
}

/** A single skeleton line */
export function SkeletonLine({ className = '' }: SkeletonProps) {
  return <div className={`h-4 bg-ic-bg-tertiary rounded animate-pulse ${className}`} />;
}

/** A wider skeleton block (e.g., for card bodies) */
export function SkeletonBlock({ className = '' }: SkeletonProps) {
  return <div className={`h-20 bg-ic-bg-tertiary rounded animate-pulse ${className}`} />;
}

/** A circular skeleton (e.g., for avatars/logos) */
export function SkeletonCircle({ className = '' }: SkeletonProps) {
  return <div className={`h-10 w-10 bg-ic-bg-tertiary rounded-full animate-pulse ${className}`} />;
}

/** Full card skeleton wrapper */
export function SkeletonCard({ className = '', lines = 3 }: SkeletonProps & { lines?: number }) {
  return (
    <div
      className={`bg-ic-surface rounded-lg border border-ic-border p-6 ${className}`}
      style={{ boxShadow: 'var(--ic-shadow-card)' }}
    >
      <div className="animate-pulse space-y-4">
        <SkeletonLine className="w-1/3 h-5" />
        {Array.from({ length: lines }).map((_, i) => (
          <div key={i} className="flex justify-between items-center">
            <SkeletonLine className="w-1/3" />
            <SkeletonLine className="w-1/4" />
          </div>
        ))}
      </div>
    </div>
  );
}
