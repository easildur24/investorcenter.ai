/**
 * Tests for ICScoreMethodology component.
 *
 * Verifies rendering of the methodology page including sidebar navigation,
 * section content display, and interactive section switching.
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';

import ICScoreMethodology from '../ic-score/ICScoreMethodology';

describe('ICScoreMethodology', () => {
  it('renders without crashing', () => {
    render(<ICScoreMethodology />);
    expect(screen.getByText('IC Score Methodology')).toBeInTheDocument();
  });

  it('renders the page subtitle', () => {
    render(<ICScoreMethodology />);
    expect(
      screen.getByText('Understanding how the Investor Center Score evaluates stocks')
    ).toBeInTheDocument();
  });

  it('renders all sidebar navigation items', () => {
    render(<ICScoreMethodology />);

    const expectedTitles = [
      'Overview',
      'Scoring Factors',
      'Categories',
      'Lifecycle Classification',
      'Sector-Relative Scoring',
      'Confidence & Data Quality',
      'Score Stability',
      'Interpretation Guide',
    ];

    expectedTitles.forEach((title) => {
      expect(screen.getByRole('button', { name: title })).toBeInTheDocument();
    });
  });

  it('displays Overview section by default', () => {
    render(<ICScoreMethodology />);
    expect(screen.getByText('What is the IC Score?')).toBeInTheDocument();
  });

  it('shows Key Features in the Overview section', () => {
    render(<ICScoreMethodology />);
    expect(screen.getByText('Key Features')).toBeInTheDocument();
    expect(screen.getByText(/Sector-Relative:/)).toBeInTheDocument();
    expect(screen.getByText(/Lifecycle-Aware:/)).toBeInTheDocument();
    expect(screen.getByText(/Multi-Factor:/)).toBeInTheDocument();
    expect(screen.getByText(/Transparent:/)).toBeInTheDocument();
  });

  it('shows Score Ratings table in the Overview section', () => {
    render(<ICScoreMethodology />);
    expect(screen.getByText('Score Ratings')).toBeInTheDocument();
    expect(screen.getByText('Strong Buy')).toBeInTheDocument();
    expect(screen.getByText('Buy')).toBeInTheDocument();
    expect(screen.getByText('Hold')).toBeInTheDocument();
    expect(screen.getByText('Sell')).toBeInTheDocument();
    expect(screen.getByText('Strong Sell')).toBeInTheDocument();
  });

  it('switches to Scoring Factors section when clicked', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Scoring Factors' }));

    // The heading "Scoring Factors" appears as both a nav button and an h2 heading
    expect(screen.getByRole('heading', { name: 'Scoring Factors' })).toBeInTheDocument();
    expect(
      screen.getByText(
        'The IC Score combines 10 individual factors, each measuring a specific aspect of stock quality, valuation, or market signals.'
      )
    ).toBeInTheDocument();
  });

  it('displays all 10 scoring factors when Scoring Factors tab is active', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Scoring Factors' }));

    const expectedFactors = [
      'Growth',
      'Profitability',
      'Financial Health',
      'Relative Value',
      'Intrinsic Value',
      'Historical Value',
      'Earnings Revisions',
      'Momentum',
      'Smart Money',
      'Technical Signals',
    ];

    expectedFactors.forEach((factor) => {
      expect(screen.getByText(factor)).toBeInTheDocument();
    });
  });

  it('displays factor categories (Quality, Valuation, Signals) in Scoring Factors', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Scoring Factors' }));

    // Multiple factors belong to each category, so use getAllByText
    expect(screen.getAllByText('Quality').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Valuation').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Signals').length).toBeGreaterThanOrEqual(1);
  });

  it('switches to Categories section and displays the three categories', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Categories' }));

    expect(screen.getByText('Score Categories')).toBeInTheDocument();
    expect(screen.getByText('~34% of total')).toBeInTheDocument();
    expect(screen.getByText('~38% of total')).toBeInTheDocument();
    expect(screen.getByText('~28% of total')).toBeInTheDocument();
  });

  it('switches to Lifecycle Classification section', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Lifecycle Classification' }));

    expect(screen.getByRole('heading', { name: 'Lifecycle Classification' })).toBeInTheDocument();

    const stageNames = ['Hypergrowth', 'Mature', 'Turnaround'];
    stageNames.forEach((stage) => {
      expect(screen.getByText(stage)).toBeInTheDocument();
    });
    // 'Growth' and 'Value' also appear as nav buttons, so use getAllByText
    expect(screen.getAllByText('Growth').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('Value').length).toBeGreaterThanOrEqual(1);
  });

  it('switches to Sector-Relative Scoring section', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Sector-Relative Scoring' }));

    expect(screen.getByRole('heading', { name: 'Sector-Relative Scoring' })).toBeInTheDocument();
    expect(screen.getByText('Why Sector-Relative?')).toBeInTheDocument();
    expect(screen.getByText('MSFT')).toBeInTheDocument();
    expect(screen.getByText('WMT')).toBeInTheDocument();
  });

  it('switches to Confidence section', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Confidence & Data Quality' }));

    expect(screen.getByRole('heading', { name: 'Confidence & Data Quality' })).toBeInTheDocument();
    expect(screen.getByText('High')).toBeInTheDocument();
    expect(screen.getByText('Medium')).toBeInTheDocument();
    expect(screen.getByText('Low')).toBeInTheDocument();
  });

  it('switches to Score Stability section', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Score Stability' }));

    expect(screen.getByRole('heading', { name: 'Score Stability' })).toBeInTheDocument();
    expect(screen.getByText('Smoothing Formula')).toBeInTheDocument();
    expect(screen.getByText('Event-Based Resets')).toBeInTheDocument();
  });

  it('switches to Interpretation Guide section', () => {
    render(<ICScoreMethodology />);

    fireEvent.click(screen.getByRole('button', { name: 'Interpretation Guide' }));

    expect(screen.getByRole('heading', { name: 'Interpretation Guide' })).toBeInTheDocument();
    expect(screen.getByText('DO use IC Score for:')).toBeInTheDocument();
    expect(screen.getByText('DO NOT use IC Score for:')).toBeInTheDocument();
    expect(screen.getByText('Important Disclaimers')).toBeInTheDocument();
  });

  it('applies active styling to the selected navigation button', () => {
    render(<ICScoreMethodology />);

    const overviewButton = screen.getByRole('button', { name: 'Overview' });
    // Overview is active by default, should have active classes
    expect(overviewButton.className).toContain('bg-blue-100');
    expect(overviewButton.className).toContain('text-blue-700');

    // Click a different section
    const factorsButton = screen.getByRole('button', { name: 'Scoring Factors' });
    fireEvent.click(factorsButton);

    // Now Scoring Factors should be active
    expect(factorsButton.className).toContain('bg-blue-100');
    expect(factorsButton.className).toContain('text-blue-700');

    // Overview should no longer be active
    expect(overviewButton.className).not.toContain('bg-blue-100');
  });

  it('hides previous section content when switching sections', () => {
    render(<ICScoreMethodology />);

    // Overview should be visible
    expect(screen.getByText('What is the IC Score?')).toBeInTheDocument();

    // Switch to Scoring Factors
    fireEvent.click(screen.getByRole('button', { name: 'Scoring Factors' }));

    // Overview content should be gone
    expect(screen.queryByText('What is the IC Score?')).not.toBeInTheDocument();
  });
});
