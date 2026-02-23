'use client';

import { useState } from 'react';
import Link from 'next/link';
import { auth } from '@/lib/api/routes';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);
  const [resetInfo, setResetInfo] = useState<{
    reset_url?: string;
    token?: string;
    expires_at?: string;
    note?: string;
  } | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess(false);
    setResetInfo(null);
    setLoading(true);

    try {
      const response = await fetch(`${API_BASE_URL}${auth.forgotPassword}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ email }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to request password reset');
      }

      setSuccess(true);
      // For development, the API returns the reset link
      if (data.reset_url) {
        setResetInfo(data);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to request password reset');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-ic-bg-secondary">
      <div className="w-full max-w-md p-8 bg-ic-surface rounded-lg shadow-md">
        <h1 className="text-2xl font-bold mb-2 text-center text-ic-text-primary">
          Forgot Password
        </h1>
        <p className="text-sm text-ic-text-muted text-center mb-6">
          Enter your email address and we&apos;ll send you instructions to reset your password.
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-red-700 rounded">
            {error}
          </div>
        )}

        {success && !resetInfo && (
          <div className="mb-4 p-3 bg-green-100 border border-green-400 text-green-700 rounded">
            <strong>Success!</strong> If an account exists with this email, you will receive
            password reset instructions.
          </div>
        )}

        {resetInfo && (
          <div className="mb-4 p-4 bg-blue-50 border border-blue-400 text-blue-900 rounded">
            <strong className="block mb-2">Development Mode - Reset Link Generated</strong>
            <p className="text-sm mb-2">{resetInfo.note}</p>
            <div className="mb-2">
              <label className="block text-xs font-semibold mb-1">Reset Link:</label>
              <a
                href={resetInfo.reset_url}
                className="text-blue-600 hover:underline text-sm break-all"
              >
                {resetInfo.reset_url}
              </a>
            </div>
            <p className="text-xs text-ic-text-muted mt-2">
              Link expires at:{' '}
              {resetInfo.expires_at ? new Date(resetInfo.expires_at).toLocaleString() : 'N/A'}
            </p>
          </div>
        )}

        {!success && (
          <form onSubmit={handleSubmit}>
            <div className="mb-6">
              <label
                className="block text-sm font-medium text-ic-text-secondary mb-2"
                htmlFor="email"
              >
                Email Address
              </label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-ic-blue text-ic-text-primary"
                placeholder="your.email@example.com"
                required
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 px-4 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed"
            >
              {loading ? 'Sending...' : 'Send Reset Instructions'}
            </button>
          </form>
        )}

        <div className="mt-6 text-center text-sm">
          <Link href="/auth/login" className="text-blue-600 hover:underline">
            Back to Login
          </Link>
        </div>

        {!success && (
          <div className="mt-2 text-center text-sm text-ic-text-muted">
            Don&apos;t have an account?{' '}
            <Link href="/auth/signup" className="text-blue-600 hover:underline">
              Sign up
            </Link>
          </div>
        )}
      </div>
    </div>
  );
}
