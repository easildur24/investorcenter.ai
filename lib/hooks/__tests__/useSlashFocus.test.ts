import { renderHook } from '@testing-library/react';
import { useRef } from 'react';
import { useSlashFocus } from '../useSlashFocus';

// Wrapper that creates a real ref and wires up the hook
function setup() {
  const focusMock = jest.fn();

  const { unmount } = renderHook(() => {
    const inputRef = useRef<HTMLInputElement>(null);

    // Simulate a DOM element with focus()
    Object.defineProperty(inputRef, 'current', {
      value: { focus: focusMock },
      writable: true,
    });

    useSlashFocus(inputRef);
    return inputRef;
  });

  return { focusMock, unmount };
}

describe('useSlashFocus', () => {
  it('focuses the input when "/" is pressed', () => {
    const { focusMock } = setup();

    const event = new KeyboardEvent('keydown', { key: '/', bubbles: true });
    document.dispatchEvent(event);

    expect(focusMock).toHaveBeenCalledTimes(1);
  });

  it('does not focus on other keys', () => {
    const { focusMock } = setup();

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', bubbles: true }));
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }));
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));

    expect(focusMock).not.toHaveBeenCalled();
  });

  it('ignores "/" when target is an INPUT element', () => {
    const { focusMock } = setup();

    const input = document.createElement('input');
    document.body.appendChild(input);
    input.focus();

    const event = new KeyboardEvent('keydown', { key: '/', bubbles: true });
    Object.defineProperty(event, 'target', { value: input });
    document.dispatchEvent(event);

    expect(focusMock).not.toHaveBeenCalled();
    document.body.removeChild(input);
  });

  it('ignores "/" when target is a TEXTAREA element', () => {
    const { focusMock } = setup();

    const textarea = document.createElement('textarea');
    document.body.appendChild(textarea);

    const event = new KeyboardEvent('keydown', { key: '/', bubbles: true });
    Object.defineProperty(event, 'target', { value: textarea });
    document.dispatchEvent(event);

    expect(focusMock).not.toHaveBeenCalled();
    document.body.removeChild(textarea);
  });

  it('ignores "/" when target is a SELECT element', () => {
    const { focusMock } = setup();

    const select = document.createElement('select');
    document.body.appendChild(select);

    const event = new KeyboardEvent('keydown', { key: '/', bubbles: true });
    Object.defineProperty(event, 'target', { value: select });
    document.dispatchEvent(event);

    expect(focusMock).not.toHaveBeenCalled();
    document.body.removeChild(select);
  });

  it('removes keydown listener on unmount', () => {
    const { focusMock, unmount } = setup();

    unmount();

    document.dispatchEvent(new KeyboardEvent('keydown', { key: '/', bubbles: true }));

    expect(focusMock).not.toHaveBeenCalled();
  });
});
