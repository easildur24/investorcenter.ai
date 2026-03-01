import Link from 'next/link';
import { ChartBarIcon } from '@heroicons/react/24/outline';

export default function Footer() {
  return (
    <footer className="bg-ic-bg-secondary">
      <div className="max-w-7xl mx-auto py-12 px-4 sm:px-6 lg:py-16 lg:px-8">
        <div className="xl:grid xl:grid-cols-3 xl:gap-8">
          <div className="space-y-8 xl:col-span-1">
            <div className="flex items-center">
              <ChartBarIcon className="h-8 w-8 text-ic-blue" />
              <span className="ml-2 text-xl font-bold text-ic-text-primary">InvestorCenter</span>
            </div>
            <p className="text-ic-text-muted text-base">
              Professional financial data and analytics platform for informed investment decisions.
            </p>
          </div>
          <div className="mt-12 grid grid-cols-2 gap-8 xl:mt-0 xl:col-span-2">
            <div className="md:grid md:grid-cols-2 md:gap-8">
              <div>
                <h3 className="text-sm font-semibold text-ic-text-dim tracking-wider uppercase">
                  Platform
                </h3>
                <ul className="mt-4 space-y-4">
                  <li>
                    <Link
                      href="/screener"
                      className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors"
                    >
                      Charts
                    </Link>
                  </li>
                  <li>
                    <Link
                      href="/screener"
                      className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors"
                    >
                      Data
                    </Link>
                  </li>
                  <li>
                    <Link
                      href="/ic-score"
                      className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors"
                    >
                      Analytics
                    </Link>
                  </li>
                </ul>
              </div>
              <div className="mt-12 md:mt-0">
                <h3 className="text-sm font-semibold text-ic-text-dim tracking-wider uppercase">
                  Company
                </h3>
                <ul className="mt-4 space-y-4">
                  <li>
                    <Link
                      href="/coming-soon"
                      className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors"
                    >
                      About
                    </Link>
                  </li>
                  <li>
                    <Link
                      href="/coming-soon"
                      className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors"
                    >
                      Contact
                    </Link>
                  </li>
                  <li>
                    <Link
                      href="/coming-soon"
                      className="text-base text-ic-text-muted hover:text-ic-text-primary transition-colors"
                    >
                      Privacy
                    </Link>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>
        <div className="mt-12 border-t border-ic-border pt-8">
          <p className="text-base text-ic-text-dim xl:text-center">
            &copy; {new Date().getFullYear()} InvestorCenter. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}
