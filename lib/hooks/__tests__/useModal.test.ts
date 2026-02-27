import { renderHook, act } from '@testing-library/react';
import { useModal } from '../useModal';

// Store the original requestAnimationFrame
const originalRAF = global.requestAnimationFrame;

beforeEach(() => {
  jest.clearAllMocks();
  // Reset body overflow
  document.body.style.overflow = '';
  // Mock requestAnimationFrame to execute callback immediately
  global.requestAnimationFrame = (cb: FrameRequestCallback) => {
    cb(0);
    return 0;
  };
});

afterEach(() => {
  global.requestAnimationFrame = originalRAF;
});

describe('useModal', () => {
  describe('body scroll lock', () => {
    it('sets body overflow to hidden on mount', () => {
      const onClose = jest.fn();
      renderHook(() => useModal(onClose));

      expect(document.body.style.overflow).toBe('hidden');
    });

    it('restores body overflow on unmount', () => {
      document.body.style.overflow = 'auto';

      const onClose = jest.fn();
      const { unmount } = renderHook(() => useModal(onClose));

      expect(document.body.style.overflow).toBe('hidden');

      unmount();

      expect(document.body.style.overflow).toBe('auto');
    });

    it('restores empty overflow if that was the previous value', () => {
      document.body.style.overflow = '';

      const onClose = jest.fn();
      const { unmount } = renderHook(() => useModal(onClose));

      expect(document.body.style.overflow).toBe('hidden');

      unmount();

      expect(document.body.style.overflow).toBe('');
    });
  });

  describe('escape key handling', () => {
    it('calls onClose when Escape is pressed', () => {
      const onClose = jest.fn();
      renderHook(() => useModal(onClose));

      act(() => {
        const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true });
        document.dispatchEvent(event);
      });

      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('does not call onClose for other keys', () => {
      const onClose = jest.fn();
      renderHook(() => useModal(onClose));

      act(() => {
        const event = new KeyboardEvent('keydown', { key: 'Enter', bubbles: true });
        document.dispatchEvent(event);
      });

      expect(onClose).not.toHaveBeenCalled();
    });

    it('stops propagation on Escape', () => {
      const onClose = jest.fn();
      renderHook(() => useModal(onClose));

      const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true });
      const stopPropSpy = jest.spyOn(event, 'stopPropagation');

      act(() => {
        document.dispatchEvent(event);
      });

      expect(stopPropSpy).toHaveBeenCalled();
    });
  });

  describe('focus trap', () => {
    it('wraps focus from last to first element on Tab', () => {
      const onClose = jest.fn();
      const { result } = renderHook(() => useModal(onClose));

      // Create a mock modal DOM structure
      const modal = document.createElement('div');
      const button1 = document.createElement('button');
      button1.textContent = 'First';
      const button2 = document.createElement('button');
      button2.textContent = 'Last';
      modal.appendChild(button1);
      modal.appendChild(button2);
      document.body.appendChild(modal);

      // Assign the modalRef current to our mock modal
      Object.defineProperty(result.current, 'current', {
        value: modal,
        writable: true,
      });

      // Focus the last element
      button2.focus();

      act(() => {
        const event = new KeyboardEvent('keydown', {
          key: 'Tab',
          bubbles: true,
        });
        // We need to verify preventDefault is called
        const preventDefaultSpy = jest.spyOn(event, 'preventDefault');
        document.dispatchEvent(event);

        // When activeElement is last and Tab is pressed (no shift),
        // it should call preventDefault and focus first
        expect(preventDefaultSpy).toHaveBeenCalled();
      });

      expect(document.activeElement).toBe(button1);

      // Cleanup
      document.body.removeChild(modal);
    });

    it('wraps focus from first to last element on Shift+Tab', () => {
      const onClose = jest.fn();
      const { result } = renderHook(() => useModal(onClose));

      const modal = document.createElement('div');
      const button1 = document.createElement('button');
      button1.textContent = 'First';
      const button2 = document.createElement('button');
      button2.textContent = 'Last';
      modal.appendChild(button1);
      modal.appendChild(button2);
      document.body.appendChild(modal);

      Object.defineProperty(result.current, 'current', {
        value: modal,
        writable: true,
      });

      // Focus the first element
      button1.focus();

      act(() => {
        const event = new KeyboardEvent('keydown', {
          key: 'Tab',
          shiftKey: true,
          bubbles: true,
        });
        const preventDefaultSpy = jest.spyOn(event, 'preventDefault');
        document.dispatchEvent(event);
        expect(preventDefaultSpy).toHaveBeenCalled();
      });

      expect(document.activeElement).toBe(button2);

      document.body.removeChild(modal);
    });

    it('does not trap focus when no focusable elements', () => {
      const onClose = jest.fn();
      const { result } = renderHook(() => useModal(onClose));

      const modal = document.createElement('div');
      modal.textContent = 'No focusable elements';
      document.body.appendChild(modal);

      Object.defineProperty(result.current, 'current', {
        value: modal,
        writable: true,
      });

      // Should not throw
      act(() => {
        const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true });
        document.dispatchEvent(event);
      });

      expect(onClose).not.toHaveBeenCalled();

      document.body.removeChild(modal);
    });

    it('does not prevent default when Tab and not on boundary', () => {
      const onClose = jest.fn();
      const { result } = renderHook(() => useModal(onClose));

      const modal = document.createElement('div');
      const button1 = document.createElement('button');
      const button2 = document.createElement('button');
      const button3 = document.createElement('button');
      modal.appendChild(button1);
      modal.appendChild(button2);
      modal.appendChild(button3);
      document.body.appendChild(modal);

      Object.defineProperty(result.current, 'current', {
        value: modal,
        writable: true,
      });

      // Focus the middle element (not first or last)
      button2.focus();

      act(() => {
        const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true });
        const preventDefaultSpy = jest.spyOn(event, 'preventDefault');
        document.dispatchEvent(event);
        // Should NOT prevent default since we're in the middle
        expect(preventDefaultSpy).not.toHaveBeenCalled();
      });

      document.body.removeChild(modal);
    });
  });

  describe('auto-focus', () => {
    it('returns a ref object', () => {
      const onClose = jest.fn();
      const { result } = renderHook(() => useModal(onClose));

      expect(result.current).toHaveProperty('current');
    });
  });

  describe('cleanup', () => {
    it('removes keydown listener on unmount', () => {
      const onClose = jest.fn();
      const { unmount } = renderHook(() => useModal(onClose));

      unmount();

      // After unmount, pressing Escape should not call onClose
      act(() => {
        const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true });
        document.dispatchEvent(event);
      });

      expect(onClose).not.toHaveBeenCalled();
    });
  });
});
