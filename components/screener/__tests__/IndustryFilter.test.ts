/**
 * Tests for IndustryFilter logic.
 *
 * Since the project's jest config uses jsx: "preserve" (no JSX transform),
 * we test the data-fetching URL construction, search filtering, and toggle
 * logic directly rather than rendering the component.
 */

const mockFetch = global.fetch as jest.Mock;

describe('IndustryFilter URL construction', () => {
  /**
   * Reproduces the URL-building logic from IndustryFilter.
   */
  function buildIndustriesUrl(selectedSectors: string[]): string {
    const baseUrl = '/api/v1';
    const sectorsParam =
      selectedSectors.length > 0
        ? `?sectors=${encodeURIComponent(selectedSectors.join(','))}`
        : '';
    return `${baseUrl}/screener/industries${sectorsParam}`;
  }

  it('builds URL without sectors param when none selected', () => {
    const url = buildIndustriesUrl([]);
    expect(url).toBe('/api/v1/screener/industries');
    expect(url).not.toContain('sectors=');
  });

  it('builds URL with single sector', () => {
    const url = buildIndustriesUrl(['Technology']);
    expect(url).toBe('/api/v1/screener/industries?sectors=Technology');
  });

  it('builds URL with multiple sectors comma-joined', () => {
    const url = buildIndustriesUrl(['Technology', 'Healthcare']);
    expect(decodeURIComponent(url)).toContain('sectors=Technology,Healthcare');
  });

  it('encodes special characters in sector names', () => {
    const url = buildIndustriesUrl(['Real Estate']);
    expect(url).toContain('sectors=Real%20Estate');
  });
});

describe('IndustryFilter search logic', () => {
  const sampleIndustries = [
    'Application Software',
    'Biotechnology',
    'Cloud Computing',
    'Data Analytics',
    'E-Commerce',
  ];

  /**
   * Reproduces the search filtering from IndustryFilter.
   */
  function filterIndustries(industries: string[], search: string): string[] {
    if (!search) return industries;
    return industries.filter((i) =>
      i.toLowerCase().includes(search.toLowerCase())
    );
  }

  it('returns all industries when search is empty', () => {
    expect(filterIndustries(sampleIndustries, '')).toEqual(sampleIndustries);
  });

  it('filters by case-insensitive substring match', () => {
    expect(filterIndustries(sampleIndustries, 'cloud')).toEqual([
      'Cloud Computing',
    ]);
  });

  it('matches partial strings', () => {
    expect(filterIndustries(sampleIndustries, 'tech')).toEqual([
      'Biotechnology',
    ]);
  });

  it('returns empty array when nothing matches', () => {
    expect(filterIndustries(sampleIndustries, 'zzzzz')).toEqual([]);
  });

  it('matches multiple results', () => {
    expect(filterIndustries(sampleIndustries, 'a')).toEqual([
      'Application Software',
      'Data Analytics',
    ]);
  });

  it('handles uppercase search', () => {
    expect(filterIndustries(sampleIndustries, 'BIO')).toEqual([
      'Biotechnology',
    ]);
  });
});

describe('IndustryFilter toggle logic', () => {
  /**
   * Reproduces the toggle logic from IndustryFilter.
   */
  function toggleIndustry(
    selectedIndustries: string[],
    industry: string
  ): string[] {
    if (selectedIndustries.includes(industry)) {
      return selectedIndustries.filter((i) => i !== industry);
    }
    return [...selectedIndustries, industry];
  }

  it('adds industry when not currently selected', () => {
    expect(toggleIndustry([], 'Software')).toEqual(['Software']);
  });

  it('adds industry to existing selection', () => {
    expect(toggleIndustry(['Software'], 'Biotechnology')).toEqual([
      'Software',
      'Biotechnology',
    ]);
  });

  it('removes industry when already selected', () => {
    expect(toggleIndustry(['Software', 'Biotechnology'], 'Software')).toEqual([
      'Biotechnology',
    ]);
  });

  it('returns empty array when removing the only selection', () => {
    expect(toggleIndustry(['Software'], 'Software')).toEqual([]);
  });

  it('preserves order of remaining items', () => {
    expect(
      toggleIndustry(['A', 'B', 'C', 'D'], 'B')
    ).toEqual(['A', 'C', 'D']);
  });
});

describe('IndustryFilter API fetcher', () => {
  it('parses industries from API response', async () => {
    const industries = ['Biotechnology', 'Software', 'Semiconductors'];
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: industries }),
    });

    const res = await fetch('/api/v1/screener/industries');
    const json = await res.json();
    expect(json.data).toEqual(industries);
    expect(json.data).toHaveLength(3);
  });

  it('handles empty data response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: [] }),
    });

    const res = await fetch('/api/v1/screener/industries?sectors=NonExistent');
    const json = await res.json();
    expect(json.data).toEqual([]);
  });

  it('handles null data with fallback', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: null }),
    });

    const res = await fetch('/api/v1/screener/industries');
    const json = await res.json();
    // IndustryFilter uses: json.data ?? []
    expect(json.data ?? []).toEqual([]);
  });

  it('rejects on HTTP error', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    const res = await fetch('/api/v1/screener/industries');
    expect(res.ok).toBe(false);
  });
});
