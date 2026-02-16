'use client';

import { useState, useCallback } from 'react';
import { API_BASE_URL } from '@/lib/api';
import type { ScreenerApiParams } from '@/lib/types/screener';

interface ExportButtonProps {
  params: ScreenerApiParams;
}

/** CSV export button that downloads filtered screener data. */
export function ExportButton({ params }: ExportButtonProps) {
  const [exporting, setExporting] = useState(false);

  const handleExport = useCallback(async () => {
    setExporting(true);
    try {
      // Build URL with same filters but no pagination
      const searchParams = new URLSearchParams();
      for (const [key, value] of Object.entries(params)) {
        if (key === 'page' || key === 'limit') continue;
        if (value !== undefined && value !== null && value !== '') {
          searchParams.append(key, String(value));
        }
      }
      const qs = searchParams.toString();
      const url = `${API_BASE_URL}/screener/stocks/export${qs ? `?${qs}` : ''}`;

      const res = await fetch(url);
      if (!res.ok) throw new Error(`Export failed: HTTP ${res.status}`);

      const blob = await res.blob();
      const downloadUrl = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = downloadUrl;
      a.download = res.headers.get('Content-Disposition')?.split('filename=')[1] ?? 'screener-export.csv';
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(downloadUrl);
    } catch (err) {
      console.error('CSV export failed:', err);
    } finally {
      setExporting(false);
    }
  }, [params]);

  return (
    <button
      onClick={handleExport}
      disabled={exporting}
      className="flex items-center gap-1.5 px-3 py-1.5 text-sm border border-ic-border rounded-md text-ic-text-secondary hover:bg-ic-surface-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      title="Export filtered results as CSV"
    >
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
          d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      {exporting ? 'Exporting...' : 'Export CSV'}
    </button>
  );
}
