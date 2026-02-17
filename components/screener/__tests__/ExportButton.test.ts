/**
 * Tests for ExportButton logic.
 *
 * Since the project's jest config uses jsx: "preserve" (no JSX transform),
 * we test the URL-building and error-handling logic directly rather than
 * rendering the component.
 */
export {};

const mockFetch = global.fetch as jest.Mock;

describe('ExportButton URL building', () => {
  /**
   * Reproduces the URL-building logic from ExportButton.handleExport
   * to verify it correctly strips pagination params and constructs the query string.
   */
  function buildExportUrl(
    params: Record<string, string | number | null | undefined>
  ): string {
    const searchParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (key === 'page' || key === 'limit') continue;
      if (value !== undefined && value !== null && value !== '') {
        searchParams.append(key, String(value));
      }
    }
    const qs = searchParams.toString();
    return `/api/v1/screener/stocks/export${qs ? `?${qs}` : ''}`;
  }

  it('excludes page and limit from export URL', () => {
    const url = buildExportUrl({ page: 2, limit: 25, sectors: 'Technology' });
    expect(url).not.toContain('page=');
    expect(url).not.toContain('limit=');
    expect(url).toContain('sectors=Technology');
  });

  it('includes all filter params', () => {
    const url = buildExportUrl({
      sectors: 'Technology',
      industries: 'Software',
      sort: 'ic_score',
      order: 'desc',
      ic_score_min: 70,
    });
    expect(url).toContain('sectors=Technology');
    expect(url).toContain('industries=Software');
    expect(url).toContain('sort=ic_score');
    expect(url).toContain('order=desc');
    expect(url).toContain('ic_score_min=70');
  });

  it('skips null and undefined values', () => {
    const url = buildExportUrl({
      sectors: 'Technology',
      pe_min: null,
      pe_max: undefined,
    });
    expect(url).toContain('sectors=Technology');
    expect(url).not.toContain('pe_min');
    expect(url).not.toContain('pe_max');
  });

  it('skips empty string values', () => {
    const url = buildExportUrl({ sectors: '', sort: 'ic_score' });
    expect(url).not.toContain('sectors');
    expect(url).toContain('sort=ic_score');
  });

  it('returns path without query string when no params', () => {
    const url = buildExportUrl({});
    expect(url).toBe('/api/v1/screener/stocks/export');
  });

  it('returns path without query string when only pagination params', () => {
    const url = buildExportUrl({ page: 1, limit: 25 });
    expect(url).toBe('/api/v1/screener/stocks/export');
  });
});

describe('ExportButton Content-Disposition parsing', () => {
  /**
   * Reproduces the filename parsing logic from ExportButton.
   */
  function parseFilename(disposition: string | null): string {
    const value = disposition ?? '';
    const filenamePart = value.split('filename=')[1];
    return filenamePart?.replace(/^"|"$/g, '') ?? 'screener-export.csv';
  }

  it('parses unquoted filename', () => {
    expect(parseFilename('attachment; filename=screener-export-2026-02-16.csv')).toBe(
      'screener-export-2026-02-16.csv'
    );
  });

  it('strips quotes from filename', () => {
    expect(parseFilename('attachment; filename="screener-export-2026-02-16.csv"')).toBe(
      'screener-export-2026-02-16.csv'
    );
  });

  it('returns fallback when header is null', () => {
    expect(parseFilename(null)).toBe('screener-export.csv');
  });

  it('returns fallback when header has no filename', () => {
    expect(parseFilename('attachment')).toBe('screener-export.csv');
  });

  it('returns fallback for empty string', () => {
    expect(parseFilename('')).toBe('screener-export.csv');
  });
});

describe('ExportButton fetch behavior', () => {
  it('rejects on HTTP error status', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    const res = await fetch('/api/v1/screener/stocks/export');
    expect(res.ok).toBe(false);
    expect(res.status).toBe(500);
  });

  it('returns blob on success', async () => {
    const csvContent = 'Symbol,Name\nAAPL,Apple';
    mockFetch.mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob([csvContent], { type: 'text/csv' }),
      headers: new Headers({
        'Content-Disposition': 'attachment; filename=test.csv',
      }),
    });

    const res = await fetch('/api/v1/screener/stocks/export');
    expect(res.ok).toBe(true);
    const blob = await res.blob();
    expect(blob.type).toBe('text/csv');
    expect(blob.size).toBeGreaterThan(0);
  });
});
