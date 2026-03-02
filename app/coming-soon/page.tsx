import Link from 'next/link';
import { SparklesIcon, ArrowLeftIcon } from '@heroicons/react/24/outline';

export default function ComingSoonPage() {
  return (
    <div className="min-h-screen bg-ic-bg-primary flex items-center justify-center px-4">
      <div className="max-w-lg w-full text-center">
        <div
          className="bg-ic-surface rounded-2xl border border-ic-border p-8 sm:p-12"
          style={{ boxShadow: 'var(--ic-shadow-card)' }}
        >
          <div className="mx-auto w-16 h-16 rounded-full bg-ic-blue/10 flex items-center justify-center mb-6">
            <SparklesIcon className="h-8 w-8 text-ic-blue" />
          </div>

          <h1 className="text-2xl sm:text-3xl font-bold text-ic-text-primary mb-3">Coming Soon</h1>

          <p className="text-ic-text-muted text-base sm:text-lg mb-8 leading-relaxed">
            This page is currently under development. We&apos;re working hard to bring you new
            features and improvements. Check back soon!
          </p>

          <Link
            href="/"
            className="inline-flex items-center gap-2 px-6 py-3 bg-ic-blue hover:bg-ic-blue-hover text-white font-medium rounded-lg transition-colors"
          >
            <ArrowLeftIcon className="h-4 w-4" />
            Back to Home
          </Link>
        </div>
      </div>
    </div>
  );
}
