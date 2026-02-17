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

  it('includes value, growth, quality, dividend, undervalued, momentum presets', () => {
    const ids = PRESET_SCREENS.map(p => p.id);
    expect(ids).toContain('value');
    expect(ids).toContain('growth');
    expect(ids).toContain('quality');
    expect(ids).toContain('dividend');
    expect(ids).toContain('undervalued');
    expect(ids).toContain('momentum');
  });

  it('value preset uses multi-factor valuation screen', () => {
    const value = PRESET_SCREENS.find(p => p.id === 'value');
    expect(value?.params.pe_max).toBe(15);
    expect(value?.params.pb_max).toBe(2);
    expect(value?.params.dividend_yield_min).toBe(2);
    expect(value?.params.de_max).toBe(1.5);
  });

  it('growth preset requires strong margins alongside growth', () => {
    const growth = PRESET_SCREENS.find(p => p.id === 'growth');
    expect(growth?.params.revenue_growth_min).toBe(20);
    expect(growth?.params.eps_growth_min).toBe(15);
    expect(growth?.params.gross_margin_min).toBe(40);
  });

  it('quality preset screens on IC Score and profitability', () => {
    const quality = PRESET_SCREENS.find(p => p.id === 'quality');
    expect(quality?.params.ic_score_min).toBe(70);
    expect(quality?.params.roe_min).toBe(15);
    expect(quality?.params.net_margin_min).toBe(10);
    expect(quality?.params.current_ratio_min).toBe(1.5);
  });

  it('dividend preset requires long dividend track record', () => {
    const dividend = PRESET_SCREENS.find(p => p.id === 'dividend');
    expect(dividend?.params.dividend_yield_min).toBe(3);
    expect(dividend?.params.consec_div_years_min).toBe(10);
    expect(dividend?.params.payout_ratio_max).toBe(75);
    expect(dividend?.params.de_max).toBe(1);
  });

  it('undervalued preset combines low valuation with quality floor', () => {
    const undervalued = PRESET_SCREENS.find(p => p.id === 'undervalued');
    expect(undervalued?.params.pe_max).toBe(15);
    expect(undervalued?.params.pb_max).toBe(1.5);
    expect(undervalued?.params.roe_min).toBe(12);
    expect(undervalued?.params.ic_score_min).toBe(50);
  });

  it('momentum preset targets high-scoring movers', () => {
    const momentum = PRESET_SCREENS.find(p => p.id === 'momentum');
    expect(momentum?.params.momentum_score_min).toBe(70);
    expect(momentum?.params.technical_score_min).toBe(70);
  });
});
