'use client';

interface PaginationProps {
  currentPage: number;
  totalPages: number;
  total: number;
  pageSize: number;
  onPageChange: (page: number) => void;
}

/** Page controls with previous/next and showing count. */
export function Pagination({ currentPage, totalPages, total, pageSize, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null;

  const from = (currentPage - 1) * pageSize + 1;
  const to = Math.min(currentPage * pageSize, total);

  return (
    <div className="px-4 py-3 border-t border-ic-border flex items-center justify-between">
      <div className="text-sm text-ic-text-muted">
        Showing {from} to {to} of {total.toLocaleString()} results
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => onPageChange(Math.max(1, currentPage - 1))}
          disabled={currentPage === 1}
          className="px-3 py-1 border border-ic-border rounded-md text-sm text-ic-text-secondary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-ic-surface-hover transition-colors"
        >
          Previous
        </button>
        <span className="px-3 py-1 text-sm text-ic-text-muted">
          Page {currentPage} of {totalPages}
        </span>
        <button
          onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
          disabled={currentPage === totalPages}
          className="px-3 py-1 border border-ic-border rounded-md text-sm text-ic-text-secondary disabled:opacity-50 disabled:cursor-not-allowed hover:bg-ic-surface-hover transition-colors"
        >
          Next
        </button>
      </div>
    </div>
  );
}
