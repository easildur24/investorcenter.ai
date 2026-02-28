'use client';

import { ReactNode } from 'react';
import { SectorPercentilesProvider } from '@/lib/contexts/SectorPercentilesContext';

interface SectorPercentilesWrapperProps {
  ticker: string;
  children: ReactNode;
}

/**
 * Client component wrapper that provides SectorPercentilesContext
 * to children in the ticker page (which is a Server Component).
 * This fetches sector percentile data once and shares it with all children.
 */
export default function SectorPercentilesWrapper({
  ticker,
  children,
}: SectorPercentilesWrapperProps) {
  return <SectorPercentilesProvider ticker={ticker}>{children}</SectorPercentilesProvider>;
}
