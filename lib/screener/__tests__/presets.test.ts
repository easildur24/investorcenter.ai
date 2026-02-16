import { PRESET_SCREENS } from '../presets';

describe('presets', () => {
  it('has unique preset IDs', () => {
    const ids = PRESET_SCREENS.map(p => p.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it('every preset has name, description, and params', () => {
    for (const preset of PRESET_SCREENS) {
      expect(preset.name.length).toBeGreaterThan(0);
      expect(preset.description.length).toBeGreaterThan(0);
      expect(Object.keys(preset.params).length).toBeGreaterThan(0);
    }
  });

  it('includes value, growth, quality presets', () => {
    const ids = PRESET_SCREENS.map(p => p.id);
    expect(ids).toContain('value');
    expect(ids).toContain('growth');
    expect(ids).toContain('quality');
  });

  it('value preset has pe_max constraint', () => {
    const value = PRESET_SCREENS.find(p => p.id === 'value');
    expect(value?.params.pe_max).toBeDefined();
    expect(value?.params.pe_max).toBeLessThanOrEqual(20);
  });

  it('growth preset has revenue_growth_min constraint', () => {
    const growth = PRESET_SCREENS.find(p => p.id === 'growth');
    expect(growth?.params.revenue_growth_min).toBeDefined();
  });
});
