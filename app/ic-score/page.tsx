import ICScoreScreener from '@/components/ic-score/ICScoreScreener';

/**
 * IC Score Stock Screener Page
 *
 * Full-page screener for filtering and discovering stocks based on IC Score.
 * Features:
 * - Filter by score range, rating, sector, market cap
 * - Sort by multiple criteria
 * - Paginated results
 * - Direct links to ticker pages
 */
export default function ICScorePage() {
  return <ICScoreScreener />;
}

// Metadata for SEO
export const metadata = {
  title: 'IC Score Stock Screener - InvestorCenter.ai',
  description:
    'Filter and discover stocks using our proprietary IC Score ranking system. Analyze 10 factors including value, growth, profitability, and sentiment.',
};
