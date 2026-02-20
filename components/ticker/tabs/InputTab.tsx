'use client';

import { useState } from 'react';

interface InputTabProps {
  symbol: string;
}

export default function InputTab({ symbol }: InputTabProps) {
  const [jsonInput, setJsonInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [existingData, setExistingData] = useState<any>(null);
  const [loadingExisting, setLoadingExisting] = useState(false);

  // Load existing data
  const loadExistingData = async () => {
    setLoadingExisting(true);
    setMessage(null);
    try {
      const response = await fetch(`/api/v1/tickers/${symbol}/keystats`);
      if (response.ok) {
        const result = await response.json();
        setExistingData(result);
        setJsonInput(JSON.stringify(result.data, null, 2));
        setMessage({ type: 'success', text: 'Loaded existing data' });
      } else if (response.status === 404) {
        setMessage({ type: 'error', text: 'No existing data found for this ticker' });
        setExistingData(null);
      } else {
        throw new Error('Failed to load data');
      }
    } catch (error) {
      console.error('Error loading data:', error);
      setMessage({ type: 'error', text: 'Failed to load existing data' });
    } finally {
      setLoadingExisting(false);
    }
  };

  // Submit data
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setMessage(null);

    try {
      // Validate JSON
      const parsedData = JSON.parse(jsonInput);

      // Send to API
      const response = await fetch(`/api/v1/tickers/${symbol}/keystats`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(parsedData),
      });

      const result = await response.json();

      if (response.ok) {
        setMessage({ type: 'success', text: 'Data saved successfully!' });
        // Refresh the page data
        setTimeout(() => {
          window.location.reload();
        }, 1500);
      } else {
        setMessage({ type: 'error', text: result.message || 'Failed to save data' });
      }
    } catch (error) {
      console.error('Error:', error);
      setMessage({
        type: 'error',
        text: error instanceof Error ? error.message : 'Invalid JSON format',
      });
    } finally {
      setLoading(false);
    }
  };

  // Delete data
  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete the fundamental data for this ticker?')) {
      return;
    }

    setLoading(true);
    setMessage(null);

    try {
      const response = await fetch(`/api/v1/tickers/${symbol}/keystats`, {
        method: 'DELETE',
      });

      const result = await response.json();

      if (response.ok) {
        setMessage({ type: 'success', text: 'Data deleted successfully!' });
        setExistingData(null);
        setJsonInput('');
      } else {
        setMessage({ type: 'error', text: result.message || 'Failed to delete data' });
      }
    } catch (error) {
      console.error('Error:', error);
      setMessage({ type: 'error', text: 'Failed to delete data' });
    } finally {
      setLoading(false);
    }
  };

  // Sample JSON template
  const loadSampleTemplate = () => {
    const sample = {
      // Income Statement (TTM)
      revenue_ttm: 416160000000,
      net_income_ttm: 112010000000,
      ebit_ttm: 133050000000,
      ebitda_ttm: 144750000000,
      eps_diluted_ttm: 7.459,

      // Income Statement (Quarterly)
      revenue_quarterly: 102470000000,
      net_income_quarterly: 27470000000,
      ebit_quarterly: 32430000000,
      ebitda_quarterly: 35550000000,

      // YoY Growth
      revenue_growth_yoy: 0.0794,
      eps_growth_yoy: 0.9114,
      ebitda_growth_yoy: 0.0939,

      // Balance Sheet
      total_assets: 359240000000,
      total_liabilities: 285510000000,
      shareholders_equity: 73730000000,
      cash_and_equivalents: 54700000000,
      total_long_term_assets: 133560000000,
      total_long_term_debt: 102820000000,
      book_value: 73730000000,

      // Cash Flow
      operating_cash_flow_ttm: 111480000000,
      investing_cash_flow_ttm: 15200000000,
      financing_cash_flow_ttm: -120690000000,
      capital_expenditures_ttm: 12720000000,
      free_cash_flow_ttm: 98770000000,

      // Ratios
      pe_ratio: 32.76,
      pb_ratio: 51.9,
      ps_ratio: 9.16,
      roe: 1.697,
      roa: 0.3235,
      roic: 0.6526,

      // Margins
      gross_margin: 0.4691,
      operating_margin: 0.3197,
      net_margin: 0.2691,

      // Shares
      shares_outstanding: 14780000000,

      // Metadata
      period_end_date: '2025-09-27',
      fiscal_year: 2025,
      fiscal_quarter: 4,
      data_source: 'Manual Input',
      calculation_notes:
        'All TTM metrics calculated from last 4 quarters. Ratios use current stock price.',
    };
    setJsonInput(JSON.stringify(sample, null, 2));
    setMessage({ type: 'success', text: 'Loaded sample template - replace with actual data' });
  };

  return (
    <div className="bg-ic-surface rounded-lg shadow p-6">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-ic-text-primary mb-2">
          Manual Fundamental Data Input
        </h2>
        <p className="text-ic-text-muted">
          Upload calculated fundamental metrics as JSON for{' '}
          <span className="font-semibold text-ic-text-primary">{symbol}</span>
        </p>
      </div>

      {/* Action Buttons */}
      <div className="flex gap-3 mb-4">
        <button
          onClick={loadExistingData}
          disabled={loadingExisting}
          className="px-4 py-2 bg-ic-bg-secondary text-ic-text-primary rounded hover:bg-ic-border transition-colors disabled:opacity-50"
        >
          {loadingExisting ? 'Loading...' : 'Load Existing Data'}
        </button>
        <button
          onClick={loadSampleTemplate}
          className="px-4 py-2 bg-ic-bg-secondary text-ic-text-primary rounded hover:bg-ic-border transition-colors"
        >
          Load Sample Template
        </button>
        {existingData && (
          <button
            onClick={handleDelete}
            disabled={loading}
            className="px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors disabled:opacity-50"
          >
            Delete Data
          </button>
        )}
      </div>

      {/* Existing Data Info */}
      {existingData && (
        <div className="mb-4 p-3 bg-blue-50 border border-blue-200 rounded text-sm">
          <p className="text-blue-800">
            <strong>Existing data found:</strong> Created{' '}
            {new Date(existingData.meta.created_at).toLocaleString()}
            {existingData.meta.updated_at !== existingData.meta.created_at &&
              `, Updated ${new Date(existingData.meta.updated_at).toLocaleString()}`}
          </p>
        </div>
      )}

      {/* Message Display */}
      {message && (
        <div
          className={`mb-4 p-4 rounded ${
            message.type === 'success'
              ? 'bg-green-50 border border-green-200 text-green-800'
              : 'bg-red-50 border border-red-200 text-red-800'
          }`}
        >
          {message.text}
        </div>
      )}

      {/* JSON Input Form */}
      <form onSubmit={handleSubmit}>
        <div className="mb-4">
          <label className="block text-sm font-medium text-ic-text-secondary mb-2">
            JSON Data (paste your calculated metrics)
          </label>
          <textarea
            value={jsonInput}
            onChange={(e) => setJsonInput(e.target.value)}
            className="w-full h-96 p-3 border border-ic-border rounded font-mono text-sm bg-ic-bg-primary text-ic-text-primary focus:outline-none focus:ring-2 focus:ring-ic-primary"
            placeholder='{"revenue_ttm": 416160000000, "net_income_ttm": 112010000000, ...}'
            required
          />
          <p className="mt-2 text-xs text-ic-text-dim">
            Paste your calculated fundamental metrics in JSON format. You can use any field names.
          </p>
        </div>

        <button
          type="submit"
          disabled={loading || !jsonInput.trim()}
          className="w-full px-6 py-3 bg-ic-primary text-white rounded-lg font-medium hover:bg-ic-primary-dark transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? 'Saving...' : 'Save Fundamental Data'}
        </button>
      </form>

      {/* Instructions */}
      <div className="mt-6 p-4 bg-ic-bg-secondary rounded">
        <h3 className="font-semibold text-ic-text-primary mb-2">Instructions:</h3>
        <ol className="list-decimal list-inside space-y-1 text-sm text-ic-text-muted">
          <li>Calculate your fundamental metrics externally (Excel, Python, etc.)</li>
          <li>Format them as JSON with any field names you want</li>
          <li>Click &quot;Load Sample Template&quot; to see an example structure</li>
          <li>Paste your JSON data and click &quot;Save&quot;</li>
          <li>The data will be stored and displayed in the Key Metrics section</li>
          <li>You can update the data anytime by submitting new JSON</li>
        </ol>
      </div>

      {/* API Info */}
      <div className="mt-4 p-4 bg-ic-bg-secondary rounded text-xs text-ic-text-dim">
        <p className="font-semibold mb-1">API Endpoints:</p>
        <code className="block mb-1">GET /api/v1/tickers/{symbol}/keystats</code>
        <code className="block mb-1">POST /api/v1/tickers/{symbol}/keystats</code>
        <code className="block">DELETE /api/v1/tickers/{symbol}/keystats</code>
      </div>
    </div>
  );
}
