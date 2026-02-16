'use client';

import { Suspense } from 'react';
import { ScreenerClient } from '@/components/screener/ScreenerClient';

export default function ScreenerPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-ic-blue"></div>
      </div>
    }>
      <ScreenerClient />
    </Suspense>
  );
}
