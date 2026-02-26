/**
 * Tests for EditAlertModal component.
 *
 * Covers rendering, form pre-population, validation, submission,
 * error handling, and close interactions.
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';

import EditAlertModal from '../alerts/EditAlertModal';
import { AlertRuleWithDetails, UpdateAlertRequest } from '@/lib/api/alerts';

// ---------------------------------------------------------------------------
// Mock alert with defaults
// ---------------------------------------------------------------------------

const makeAlert = (overrides: Partial<AlertRuleWithDetails> = {}): AlertRuleWithDetails => ({
  id: 'alert-1',
  user_id: 'user-1',
  watch_list_id: 'wl-1',
  symbol: 'AAPL',
  alert_type: 'price_above',
  conditions: { threshold: 150 },
  is_active: true,
  frequency: 'daily' as const,
  notify_email: true,
  notify_in_app: true,
  name: 'AAPL Price Above $150',
  trigger_count: 0,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  watch_list_name: 'Tech Stocks',
  ...overrides,
});

const defaultProps = {
  alert: makeAlert(),
  onSave: jest.fn(),
  onClose: jest.fn(),
};

const renderModal = (overrides: Partial<typeof defaultProps> = {}) => {
  return render(<EditAlertModal {...defaultProps} {...overrides} />);
};

/** Helper: submit the form via the <form> element. */
const submitForm = () => {
  const form = screen.getByText('Update Alert').closest('form')!;
  fireEvent.submit(form);
};

// ---------------------------------------------------------------------------
// Rendering & pre-population
// ---------------------------------------------------------------------------

describe('EditAlertModal — Rendering', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders modal title', () => {
    renderModal();
    expect(screen.getByText('Edit Alert')).toBeInTheDocument();
  });

  it('shows symbol and type in subtitle', () => {
    renderModal();
    // The subtitle renders "AAPL — Price Above" with an mdash entity
    const subtitle = screen.getByText(/AAPL/);
    expect(subtitle).toBeInTheDocument();
    expect(subtitle.textContent).toContain('Price Above');
  });

  it('pre-populates threshold from conditions', () => {
    renderModal();
    const input = screen.getByLabelText(/Price threshold/) as HTMLInputElement;
    expect(input.value).toBe('150');
  });

  it('pre-populates volume_multiplier for volume_spike type', () => {
    renderModal({
      alert: makeAlert({
        alert_type: 'volume_spike',
        conditions: { volume_multiplier: 3, baseline: 'avg_30d' },
      }),
    });
    const input = screen.getByLabelText(/Volume multiplier/) as HTMLInputElement;
    expect(input.value).toBe('3');
  });

  it('pre-populates frequency', () => {
    renderModal();
    const select = screen.getByLabelText('Frequency') as HTMLSelectElement;
    expect(select.value).toBe('daily');
  });

  it('pre-populates name', () => {
    renderModal();
    const input = screen.getByLabelText('Name') as HTMLInputElement;
    expect(input.value).toBe('AAPL Price Above $150');
  });

  it('pre-populates notification checkboxes', () => {
    renderModal({ alert: makeAlert({ notify_email: false, notify_in_app: true }) });
    expect(screen.getByLabelText('Email')).not.toBeChecked();
    expect(screen.getByLabelText('In-App')).toBeChecked();
  });

  it('renders alert type as disabled select', () => {
    renderModal();
    const typeSelect = screen.getByLabelText('Alert Type') as HTMLSelectElement;
    expect(typeSelect).toBeDisabled();
  });

  it('shows hint about type immutability', () => {
    renderModal();
    expect(screen.getByText(/To change type, delete and recreate/)).toBeInTheDocument();
  });

  it('renders Update Alert button', () => {
    renderModal();
    expect(screen.getByText('Update Alert')).toBeInTheDocument();
  });

  it('renders Cancel button', () => {
    renderModal();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('renders Close (X) button', () => {
    renderModal();
    expect(screen.getByLabelText('Close')).toBeInTheDocument();
  });

  it('hides threshold input for event-based alert types', () => {
    renderModal({ alert: makeAlert({ alert_type: 'news', conditions: {} }) });
    expect(screen.queryByLabelText(/threshold/i)).not.toBeInTheDocument();
  });

  it('shows autoName as placeholder when name is editable', () => {
    renderModal();
    const input = screen.getByLabelText('Name') as HTMLInputElement;
    // The placeholder should be the auto-generated name
    expect(input.placeholder).toContain('AAPL');
  });
});

// ---------------------------------------------------------------------------
// Threshold labels
// ---------------------------------------------------------------------------

describe('EditAlertModal — Threshold labels', () => {
  beforeEach(() => jest.clearAllMocks());

  it('shows "Price threshold ($)" for price_above', () => {
    renderModal();
    expect(screen.getByText(/Price threshold \(\$\)/)).toBeInTheDocument();
  });

  it('shows "Volume multiplier" for volume_spike', () => {
    renderModal({
      alert: makeAlert({
        alert_type: 'volume_spike',
        conditions: { volume_multiplier: 2 },
      }),
    });
    expect(screen.getByText(/Volume multiplier/)).toBeInTheDocument();
  });

  it('shows "Volume threshold" for volume_above', () => {
    renderModal({
      alert: makeAlert({
        alert_type: 'volume_above',
        conditions: { threshold: 1000000 },
      }),
    });
    expect(screen.getByText(/Volume threshold/)).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

describe('EditAlertModal — Validation', () => {
  beforeEach(() => jest.clearAllMocks());

  it('shows error for empty threshold', async () => {
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid threshold value')).toBeInTheDocument();
    });
    expect(defaultProps.onSave).not.toHaveBeenCalled();
  });

  it('shows error for zero threshold', async () => {
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '0' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid threshold value')).toBeInTheDocument();
    });
    expect(defaultProps.onSave).not.toHaveBeenCalled();
  });

  it('shows error for zero volume_spike multiplier', async () => {
    renderModal({
      alert: makeAlert({
        alert_type: 'volume_spike',
        conditions: { volume_multiplier: 2 },
      }),
    });
    fireEvent.change(screen.getByLabelText(/Volume multiplier/), { target: { value: '0' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid threshold value')).toBeInTheDocument();
    });
    expect(defaultProps.onSave).not.toHaveBeenCalled();
  });

  it('shows error for negative volume_spike multiplier', async () => {
    renderModal({
      alert: makeAlert({
        alert_type: 'volume_spike',
        conditions: { volume_multiplier: 2 },
      }),
    });
    fireEvent.change(screen.getByLabelText(/Volume multiplier/), { target: { value: '-1' } });
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid threshold value')).toBeInTheDocument();
    });
    expect(defaultProps.onSave).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// Submission
// ---------------------------------------------------------------------------

describe('EditAlertModal — Submission', () => {
  beforeEach(() => jest.clearAllMocks());

  it('calls onSave with correct UpdateAlertRequest', async () => {
    defaultProps.onSave.mockResolvedValueOnce(undefined);
    renderModal();
    submitForm();

    await waitFor(() => {
      expect(defaultProps.onSave).toHaveBeenCalledWith('alert-1', {
        name: 'AAPL Price Above $150',
        conditions: { threshold: 150 },
        frequency: 'daily',
        notify_email: true,
        notify_in_app: true,
      });
    });
  });

  it('sends volume_spike conditions with volume_multiplier', async () => {
    defaultProps.onSave.mockResolvedValueOnce(undefined);
    renderModal({
      alert: makeAlert({
        alert_type: 'volume_spike',
        conditions: { volume_multiplier: 2.5 },
        name: 'AAPL Volume Spike',
      }),
    });
    submitForm();

    await waitFor(() => {
      expect(defaultProps.onSave).toHaveBeenCalledWith(
        'alert-1',
        expect.objectContaining({
          conditions: { volume_multiplier: 2.5 },
        })
      );
    });
  });

  it('reflects edited threshold in submission', async () => {
    defaultProps.onSave.mockResolvedValueOnce(undefined);
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '200' } });
    submitForm();

    await waitFor(() => {
      expect(defaultProps.onSave).toHaveBeenCalledWith(
        'alert-1',
        expect.objectContaining({ conditions: { threshold: 200 } })
      );
    });
  });

  it('reflects changed frequency', async () => {
    defaultProps.onSave.mockResolvedValueOnce(undefined);
    renderModal();
    fireEvent.change(screen.getByLabelText('Frequency'), { target: { value: 'once' } });
    submitForm();

    await waitFor(() => {
      expect(defaultProps.onSave).toHaveBeenCalledWith(
        'alert-1',
        expect.objectContaining({ frequency: 'once' })
      );
    });
  });

  it('uses auto-generated name when name is cleared', async () => {
    defaultProps.onSave.mockResolvedValueOnce(undefined);
    renderModal();
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: '' } });
    submitForm();

    await waitFor(() => {
      expect(defaultProps.onSave).toHaveBeenCalledWith(
        'alert-1',
        expect.objectContaining({ name: 'AAPL Price Above $150' })
      );
    });
  });

  it('shows "Saving..." while submitting', async () => {
    let resolvePromise: () => void;
    const promise = new Promise<void>((resolve) => {
      resolvePromise = resolve;
    });
    defaultProps.onSave.mockReturnValueOnce(promise);

    renderModal();
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Saving...')).toBeInTheDocument();
    });

    // Resolve to clean up
    resolvePromise!();
    await waitFor(() => {
      expect(screen.queryByText('Saving...')).not.toBeInTheDocument();
    });
  });

  it('disables both buttons while saving', async () => {
    let resolvePromise: () => void;
    const promise = new Promise<void>((resolve) => {
      resolvePromise = resolve;
    });
    defaultProps.onSave.mockReturnValueOnce(promise);

    renderModal();
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Saving...')).toBeInTheDocument();
    });

    // Both Update Alert (now "Saving...") and Cancel should be disabled
    expect(screen.getByText('Saving...').closest('button')).toBeDisabled();
    expect(screen.getByText('Cancel').closest('button')).toBeDisabled();

    // Resolve to clean up
    resolvePromise!();
    await waitFor(() => {
      expect(screen.queryByText('Saving...')).not.toBeInTheDocument();
    });
  });
});

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

describe('EditAlertModal — Error handling', () => {
  beforeEach(() => jest.clearAllMocks());

  it('displays error from onSave rejection', async () => {
    defaultProps.onSave.mockRejectedValueOnce(new Error('Server error'));
    renderModal();
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Server error')).toBeInTheDocument();
    });
  });

  it('displays fallback error for non-Error throws', async () => {
    defaultProps.onSave.mockRejectedValueOnce('some string');
    renderModal();
    submitForm();

    await waitFor(() => {
      expect(screen.getByText('Failed to update alert')).toBeInTheDocument();
    });
  });
});

// ---------------------------------------------------------------------------
// Close interactions
// ---------------------------------------------------------------------------

describe('EditAlertModal — Close interactions', () => {
  beforeEach(() => jest.clearAllMocks());

  it('calls onClose when Cancel is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByText('Cancel'));
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('calls onClose when X button is clicked', () => {
    renderModal();
    fireEvent.click(screen.getByLabelText('Close'));
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('calls onClose on Escape key', () => {
    renderModal();
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('calls onClose when background overlay is clicked', () => {
    renderModal();
    const overlay = screen.getByTestId('modal-overlay');
    fireEvent.click(overlay);
    expect(defaultProps.onClose).toHaveBeenCalled();
  });
});
