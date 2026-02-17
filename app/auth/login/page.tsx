'use client';

import { useState, useEffect } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useAuth } from '@/lib/auth/AuthContext';

export default function LoginPage() {
  const { login } = useAuth();
  const searchParams = useSearchParams();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [sessionExpired, setSessionExpired] = useState(false);

  useEffect(() => {
    // Check if user was redirected due to session expiration
    if (searchParams.get('session_expired') === 'true') {
      setSessionExpired(true);
    }
  }, [searchParams]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSessionExpired(false); // Clear session expired message on new login attempt
    setLoading(true);

    try {
      await login(email, password);
    } catch (err: any) {
      setError(err.message || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-ic-bg-primary">
      <div className="w-full max-w-md p-8 bg-ic-surface rounded-lg border border-ic-border">
        <h1 className="text-2xl font-bold mb-6 text-center text-ic-text-primary">
          Login to InvestorCenter.ai
        </h1>

        {sessionExpired && (
          <div className="mb-4 p-3 bg-yellow-50 border border-yellow-400 text-yellow-800 rounded">
            <strong>Session Expired:</strong> Your session has expired. Please log in again to
            continue.
          </div>
        )}

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-ic-negative rounded">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label
              className="block text-sm font-medium text-ic-text-secondary mb-2"
              htmlFor="email"
            >
              Email
            </label>
            <input
              id="email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              required
            />
          </div>

          <div className="mb-6">
            <label
              className="block text-sm font-medium text-ic-text-secondary mb-2"
              htmlFor="password"
            >
              Password
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              required
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed"
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <div className="mt-4 text-center text-sm">
          <Link href="/auth/forgot-password" className="text-ic-blue hover:underline">
            Forgot password?
          </Link>
        </div>

        <div className="mt-2 text-center text-sm text-ic-text-secondary">
          Don&apos;t have an account?{' '}
          <Link href="/auth/signup" className="text-ic-blue hover:underline">
            Sign up
          </Link>
        </div>
      </div>
    </div>
  );
}
