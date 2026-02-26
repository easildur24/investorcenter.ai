import { redirect } from 'next/navigation';

/**
 * /alerts — permanent redirect to /watchlist.
 *
 * Alerts are managed per-watchlist (Phase 3). This server-side redirect
 * ensures old bookmarks and external links still work.
 */
export default function AlertsPage() {
  redirect('/watchlist');
}
