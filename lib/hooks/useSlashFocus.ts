import { useEffect, type RefObject } from 'react';

/**
 * Keyboard shortcut hook: pressing "/" anywhere on the page focuses the target input,
 * unless the user is already typing in an editable element.
 *
 * Usage:
 *   const inputRef = useRef<HTMLInputElement>(null);
 *   useSlashFocus(inputRef);
 */
export function useSlashFocus(inputRef: RefObject<HTMLInputElement | null>) {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't capture "/" when user is typing in an input, textarea, select, or contenteditable
      const target = e.target as HTMLElement;
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return;
      }

      if (e.key === '/') {
        e.preventDefault();
        inputRef.current?.focus();
      }
    };

    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [inputRef]);
}
