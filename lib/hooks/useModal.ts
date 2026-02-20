import { useEffect, useRef, useCallback } from 'react';

/**
 * Hook that provides standard modal accessibility behavior:
 * - Escape key to dismiss
 * - Body scroll lock while open
 * - Focus trap (Tab / Shift+Tab cycle within modal)
 * - Auto-focus on the first focusable element
 *
 * Usage:
 *   const modalRef = useModal(onClose);
 *   return <div ref={modalRef} role="dialog" aria-modal="true">...</div>;
 */
export function useModal(onClose: () => void) {
  const modalRef = useRef<HTMLDivElement>(null);

  // Escape key dismiss + focus trap
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
        return;
      }

      // Focus trap: cycle Tab within modal
      if (e.key === 'Tab' && modalRef.current) {
        const focusable = modalRef.current.querySelectorAll<HTMLElement>(
          'a[href], button:not([disabled]), textarea, input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
        );
        if (focusable.length === 0) return;

        const first = focusable[0];
        const last = focusable[focusable.length - 1];

        if (e.shiftKey) {
          if (document.activeElement === first) {
            e.preventDefault();
            last.focus();
          }
        } else {
          if (document.activeElement === last) {
            e.preventDefault();
            first.focus();
          }
        }
      }
    },
    [onClose]
  );

  useEffect(() => {
    // Lock body scroll
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    // Listen for keyboard events
    document.addEventListener('keydown', handleKeyDown);

    // Auto-focus first focusable element
    if (modalRef.current) {
      const firstFocusable = modalRef.current.querySelector<HTMLElement>(
        'input:not([disabled]), textarea:not([disabled]), select:not([disabled]), button:not([disabled])'
      );
      if (firstFocusable) {
        // Delay to ensure modal is fully rendered
        requestAnimationFrame(() => firstFocusable.focus());
      }
    }

    return () => {
      document.body.style.overflow = prevOverflow;
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown]);

  return modalRef;
}
