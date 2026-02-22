'use client';

import Link from 'next/link';
import { ChartBarIcon, BellIcon } from '@heroicons/react/24/outline';
import TickerSearch from '@/components/TickerSearch';
import { useAuth } from '@/lib/auth/AuthContext';
import { ThemeToggle } from '@/lib/contexts/ThemeContext';
import { useState, useEffect } from 'react';

export default function Header() {
  const { user, logout } = useAuth();
  const [showDropdown, setShowDropdown] = useState(false);

  return (
    <nav className="bg-ic-header-bg backdrop-blur-md border-b border-ic-border">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <div className="flex-shrink-0 flex items-center">
              <Link href="/" className="flex items-center">
                <ChartBarIcon className="h-8 w-8 text-ic-blue" />
                <span className="ml-2 text-xl font-bold text-ic-text-primary">InvestorCenter</span>
              </Link>
            </div>

            {/* Navigation Links */}
            <div className="hidden md:ml-6 md:flex md:space-x-8">
              <Link
                href="/"
                className="text-ic-text-muted hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
              >
                Home
              </Link>
              <Link
                href="/screener"
                className="text-ic-text-muted hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
              >
                Screener
              </Link>
              <Link
                href="/crypto"
                className="text-ic-text-muted hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
              >
                Crypto
              </Link>
              <Link
                href="/reddit"
                className="text-ic-text-muted hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
              >
                Reddit Trends
              </Link>
              {user && (
                <>
                  <Link
                    href="/alerts"
                    className="text-ic-text-muted hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
                  >
                    Alerts
                  </Link>
                </>
              )}
            </div>
          </div>

          <div className="flex items-center space-x-2 sm:space-x-4">
            {/* Ticker Search - Always visible on all screen sizes */}
            <div className="w-36 sm:w-auto">
              <TickerSearch />
            </div>

            {/* Theme Toggle */}
            <ThemeToggle />

            {/* Alerts Bell Icon - Only when logged in */}
            {user && (
              <Link
                href="/alerts"
                className="relative p-2 text-ic-text-muted hover:text-ic-text-primary rounded-full hover:bg-ic-surface transition-colors"
                title="Alerts"
              >
                <BellIcon className="h-6 w-6" />
                {/* TODO: Add notification count badge here */}
              </Link>
            )}

            {user ? (
              <div className="relative">
                <button
                  onClick={() => setShowDropdown(!showDropdown)}
                  className="flex items-center gap-2 text-ic-text-secondary hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
                >
                  <span>{user.full_name}</span>
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 9l-7 7-7-7"
                    />
                  </svg>
                </button>

                {showDropdown && (
                  <div className="absolute right-0 mt-2 w-48 bg-ic-bg-primary rounded-md shadow-lg py-1 z-10 border border-ic-border">
                    <Link
                      href="/watchlist"
                      className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                      onClick={() => setShowDropdown(false)}
                    >
                      My Watch Lists
                    </Link>
                    <Link
                      href="/alerts"
                      className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                      onClick={() => setShowDropdown(false)}
                    >
                      My Alerts
                    </Link>
                    <Link
                      href="/settings/profile"
                      className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                      onClick={() => setShowDropdown(false)}
                    >
                      Settings
                    </Link>
                    {user.is_worker && (
                      <>
                        <div className="border-t border-ic-border my-1"></div>
                        <Link
                          href="/worker/dashboard"
                          className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                          onClick={() => setShowDropdown(false)}
                        >
                          My Tasks
                        </Link>
                      </>
                    )}
                    {user.is_admin && (
                      <>
                        <div className="border-t border-ic-border my-1"></div>
                        <div className="px-4 py-2 text-xs font-semibold text-ic-text-dim uppercase">
                          Admin
                        </div>
                        <Link
                          href="/admin/dashboard"
                          className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                          onClick={() => setShowDropdown(false)}
                        >
                          Dashboard
                        </Link>
                        <Link
                          href="/admin/cronjobs"
                          className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                          onClick={() => setShowDropdown(false)}
                        >
                          Cronjob Monitoring
                        </Link>
                        <Link
                          href="/admin/ic-scores"
                          className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                          onClick={() => setShowDropdown(false)}
                        >
                          IC Score Admin
                        </Link>
                        <Link
                          href="/admin/notes"
                          className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                          onClick={() => setShowDropdown(false)}
                        >
                          Feature Notes
                        </Link>
                        <Link
                          href="/admin/workers"
                          className="block px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                          onClick={() => setShowDropdown(false)}
                        >
                          Workers & Tasks
                        </Link>
                      </>
                    )}
                    <div className="border-t border-ic-border my-1"></div>
                    <button
                      onClick={() => {
                        setShowDropdown(false);
                        logout();
                      }}
                      className="block w-full text-left px-4 py-2 text-ic-text-secondary hover:bg-ic-surface hover:text-ic-text-primary transition-colors"
                    >
                      Logout
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <>
                <Link
                  href="/auth/login"
                  className="text-ic-text-muted hover:text-ic-text-primary px-3 py-2 rounded-md text-sm font-medium transition-colors"
                >
                  Login
                </Link>
                <Link
                  href="/auth/signup"
                  className="bg-ic-blue hover:bg-ic-blue-hover text-ic-text-primary px-4 py-2 rounded-md text-sm font-medium transition-colors"
                >
                  Get Started
                </Link>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
