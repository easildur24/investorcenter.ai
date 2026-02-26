/**
 * Tests for BulkAlertModal component.
 *
 * Covers rendering, form validation, submission, error handling,
 * threshold label switching, and user interactions (close, escape).
 */

import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';

// Mock the alerts API
jest.mock('@/lib/api/alerts', () => {
  const actual = jest.requireActual('@/lib/api/alerts');
  return {
    ...actual,
    alertAPI: {
      ...actual.alertAPI,
      bulkCreateAlerts: jest.fn(),
    },
  };
});

import BulkAlertModal from '../watchlist/BulkAlertModal';
import { alertAPI } from '@/lib/api/alerts';

const mockBulkCreate = alertAPI.bulkCreateAlerts as jest.Mock;

// ---------------------------------------------------------------------------
// Default props
// ---------------------------------------------------------------------------

const defaultProps = {
  watchListId: 'wl-123',
  watchListName: 'Tech Stocks',
  tickerCount: 5,
  onClose: jest.fn(),
  onSuccess: jest.fn(),
};

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const renderModal = (overrides: Partial<typeof defaultProps> = {}) => {
  return render(<BulkAlertModal {...defaultProps} {...overrides} />);
};

// ---------------------------------------------------------------------------
// Rendering tests
// ---------------------------------------------------------------------------

describe('BulkAlertModal — Rendering', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders modal title', () => {
    renderModal();
    expect(screen.getByText('Set Alert for All Tickers')).toBeInTheDocument();
  });

  it('shows watchlist name and ticker count in subtitle', () => {
    renderModal();
    expect(screen.getByText(/Apply to 5 tickers in/)).toBeInTheDocument();
    expect(screen.getByText(/Tech Stocks/)).toBeInTheDocument();
  });

  it('shows singular "ticker" for count of 1', () => {
    renderModal({ tickerCount: 1 });
    expect(screen.getByText(/Apply to 1 ticker in/)).toBeInTheDocument();
  });

  it('renders submit button with ticker count', () => {
    renderModal();
    expect(screen.getByText('Create Alerts (5)')).toBeInTheDocument();
  });

  it('renders Cancel button', () => {
    renderModal();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('renders Close (X) button with aria-label', () => {
    renderModal();
    expect(screen.getByLabelText('Close')).toBeInTheDocument();
  });

  it('renders all 4 bulk alert type options', () => {
    renderModal();
    const select = screen.getByLabelText('Alert Type');
    expect(select).toBeInTheDocument();

    const options = select.querySelectorAll('option');
    expect(options).toHaveLength(4);

    const optionTexts = Array.from(options).map((o) => o.textContent);
    expect(optionTexts.some((t) => t?.includes('Price Above'))).toBe(true);
    expect(optionTexts.some((t) => t?.includes('Price Below'))).toBe(true);
    expect(optionTexts.some((t) => t?.includes('Volume Above'))).toBe(true);
    expect(optionTexts.some((t) => t?.includes('Volume Spike'))).toBe(true);
  });

  it('renders frequency select with all 3 options', () => {
    renderModal();
    const select = screen.getByLabelText('Frequency');
    const options = select.querySelectorAll('option');
    expect(options).toHaveLength(3);
  });

  it('renders Email and In-App notification checkboxes (both checked by default)', () => {
    renderModal();
    const emailCheckbox = screen.getByLabelText('Email');
    const inAppCheckbox = screen.getByLabelText('In-App');

    expect(emailCheckbox).toBeChecked();
    expect(inAppCheckbox).toBeChecked();
  });

  it('renders info note about skipping and auto-generated names', () => {
    renderModal();
    expect(
      screen.getByText(/Tickers that already have an active alert will be skipped/)
    ).toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Threshold label switching
// ---------------------------------------------------------------------------

describe('BulkAlertModal — Threshold labels', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('shows "Price threshold ($)" for price_above (default)', () => {
    renderModal();
    expect(screen.getByText(/Price threshold \(\$\)/)).toBeInTheDocument();
  });

  it('shows "Price threshold ($)" for price_below', () => {
    renderModal();
    fireEvent.change(screen.getByLabelText('Alert Type'), { target: { value: 'price_below' } });
    expect(screen.getByText(/Price threshold \(\$\)/)).toBeInTheDocument();
  });

  it('shows "Volume threshold" for volume_above', () => {
    renderModal();
    fireEvent.change(screen.getByLabelText('Alert Type'), { target: { value: 'volume_above' } });
    expect(screen.getByText(/Volume threshold/)).toBeInTheDocument();
  });

  it('shows "Volume multiplier" for volume_spike', () => {
    renderModal();
    fireEvent.change(screen.getByLabelText('Alert Type'), { target: { value: 'volume_spike' } });
    expect(screen.getByText(/Volume multiplier/)).toBeInTheDocument();
  });

  it('shows 30-day average helper text for volume_spike', () => {
    renderModal();
    fireEvent.change(screen.getByLabelText('Alert Type'), { target: { value: 'volume_spike' } });
    expect(
      screen.getByText(/Triggers when volume exceeds this multiple of the 30-day average/)
    ).toBeInTheDocument();
  });

  it('does not show 30-day helper text for non-volume_spike types', () => {
    renderModal();
    expect(
      screen.queryByText(/Triggers when volume exceeds this multiple of the 30-day average/)
    ).not.toBeInTheDocument();
  });
});

// ---------------------------------------------------------------------------
// Form validation
// ---------------------------------------------------------------------------

describe('BulkAlertModal — Validation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('shows error when submitting without threshold', async () => {
    renderModal();
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Please enter a threshold value')).toBeInTheDocument();
    });

    expect(mockBulkCreate).not.toHaveBeenCalled();
  });

  it('shows error for zero threshold', async () => {
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '0' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid positive number')).toBeInTheDocument();
    });

    expect(mockBulkCreate).not.toHaveBeenCalled();
  });

  it('shows error for negative threshold', async () => {
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '-5' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Please enter a valid positive number')).toBeInTheDocument();
    });

    expect(mockBulkCreate).not.toHaveBeenCalled();
  });

  it('shows error for non-numeric threshold (type=number sanitizes to empty)', async () => {
    // jsdom sanitizes non-numeric values in type="number" inputs to empty string,
    // so the validation hits the "empty" check rather than the NaN check.
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: 'abc' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Please enter a threshold value')).toBeInTheDocument();
    });

    expect(mockBulkCreate).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// Successful submission
// ---------------------------------------------------------------------------

describe('BulkAlertModal — Submission', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('calls bulkCreateAlerts with correct payload on submit', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 5, skipped: 0 });

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '150' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(mockBulkCreate).toHaveBeenCalledWith({
        watch_list_id: 'wl-123',
        alert_type: 'price_above',
        conditions: { threshold: 150 },
        frequency: 'daily',
        notify_email: true,
        notify_in_app: true,
      });
    });
  });

  it('shows "Creating..." while loading', async () => {
    // Use a promise that we can resolve manually
    let resolvePromise: (value: any) => void;
    const promise = new Promise((resolve) => {
      resolvePromise = resolve;
    });
    mockBulkCreate.mockReturnValueOnce(promise);

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Creating...')).toBeInTheDocument();
    });

    // Resolve to clean up
    await act(async () => {
      resolvePromise!({ created: 1, skipped: 0 });
    });
  });

  it('calls onSuccess with created/skipped summary message', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 3, skipped: 2 });

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '200' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(defaultProps.onSuccess).toHaveBeenCalledWith(
        'Created 3 alerts, skipped 2 (already have alerts)'
      );
    });
  });

  it('calls onSuccess with singular "alert" for created=1', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 1, skipped: 0 });

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(defaultProps.onSuccess).toHaveBeenCalledWith('Created 1 alert');
    });
  });

  it('calls onSuccess with "No changes made" when created=0 and skipped=0', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 0, skipped: 0 });

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(defaultProps.onSuccess).toHaveBeenCalledWith('No changes made');
    });
  });

  it('sends volume_spike conditions with volume_multiplier', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 5, skipped: 0 });

    renderModal();
    fireEvent.change(screen.getByLabelText('Alert Type'), { target: { value: 'volume_spike' } });
    fireEvent.change(screen.getByLabelText(/Volume multiplier/), { target: { value: '2' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(mockBulkCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          alert_type: 'volume_spike',
          conditions: { volume_multiplier: 2, baseline: 'avg_30d' },
        })
      );
    });
  });

  it('respects unchecked notification checkboxes', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 5, skipped: 0 });

    renderModal();
    fireEvent.click(screen.getByLabelText('Email'));
    fireEvent.click(screen.getByLabelText('In-App'));
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(mockBulkCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          notify_email: false,
          notify_in_app: false,
        })
      );
    });
  });

  it('changes frequency when user selects a different option', async () => {
    mockBulkCreate.mockResolvedValueOnce({ created: 5, skipped: 0 });

    renderModal();
    fireEvent.change(screen.getByLabelText('Frequency'), { target: { value: 'once' } });
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(mockBulkCreate).toHaveBeenCalledWith(
        expect.objectContaining({
          frequency: 'once',
        })
      );
    });
  });
});

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

describe('BulkAlertModal — Error handling', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('displays API error message', async () => {
    mockBulkCreate.mockRejectedValueOnce(new Error('Alert limit reached'));

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Alert limit reached')).toBeInTheDocument();
    });

    expect(defaultProps.onSuccess).not.toHaveBeenCalled();
  });

  it('displays fallback error for non-Error throws', async () => {
    mockBulkCreate.mockRejectedValueOnce('some string error');

    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Failed to create alerts')).toBeInTheDocument();
    });
  });

  it('clears previous error on new submission', async () => {
    // First: fail
    mockBulkCreate.mockRejectedValueOnce(new Error('Some error'));
    renderModal();
    fireEvent.change(screen.getByLabelText(/Price threshold/), { target: { value: '100' } });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.getByText('Some error')).toBeInTheDocument();
    });

    // Second: succeed
    mockBulkCreate.mockResolvedValueOnce({ created: 5, skipped: 0 });
    fireEvent.submit(screen.getByText('Create Alerts (5)'));

    await waitFor(() => {
      expect(screen.queryByText('Some error')).not.toBeInTheDocument();
    });
  });
});

// ---------------------------------------------------------------------------
// Close / dismiss interactions
// ---------------------------------------------------------------------------

describe('BulkAlertModal — Close interactions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

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
    const { container } = renderModal();
    // The overlay is the first fixed inset-0 div inside the modal
    const overlay = container.querySelector('.fixed.inset-0.transition-opacity');
    expect(overlay).toBeTruthy();
    fireEvent.click(overlay!);
    expect(defaultProps.onClose).toHaveBeenCalled();
  });
});
