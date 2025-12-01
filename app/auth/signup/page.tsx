'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth/AuthContext';

export default function SignupPage() {
  const { signup } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [fullName, setFullName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password.length < 8) {
      setError('Password must be at least 8 characters');
      return;
    }

    setLoading(true);

    try {
      await signup(email, password, fullName);
    } catch (err: any) {
      setError(err.message || 'Signup failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-ic-bg-primary">
      <div className="w-full max-w-md p-8 bg-ic-surface rounded-lg border border-ic-border">
        <h1 className="text-2xl font-bold mb-6 text-center text-ic-text-primary">Create Account</h1>

        {error && (
          <div className="mb-4 p-3 bg-red-100 border border-red-400 text-ic-negative rounded">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium text-ic-text-secondary mb-2" htmlFor="fullName">
              Full Name
            </label>
            <input
              id="fullName"
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-ic-text-secondary mb-2" htmlFor="email">
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
            <label className="block text-sm font-medium text-ic-text-secondary mb-2" htmlFor="password">
              Password (min 8 characters)
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-ic-text-primary"
              required
              minLength={8}
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2 px-4 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed"
          >
            {loading ? 'Creating account...' : 'Sign Up'}
          </button>
        </form>

        <div className="mt-4 text-center text-sm text-ic-text-secondary">
          Already have an account?{' '}
          <Link href="/auth/login" className="text-ic-blue hover:underline">
            Login
          </Link>
        </div>
      </div>
    </div>
  );
}
