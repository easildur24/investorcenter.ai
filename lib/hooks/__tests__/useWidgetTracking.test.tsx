import { render, act } from '@testing-library/react';
import React from 'react';
import { analytics } from '@/lib/analytics';
import { useWidgetTracking } from '../useWidgetTracking';

// Mock analytics
jest.mock('@/lib/analytics', () => ({
  analytics: {
    track: jest.fn(),
  },
}));

const mockTrack = analytics.track as jest.Mock;

// ── IntersectionObserver mock ──────────────────────────────────────────────
type IntersectionCallback = (entries: Partial<IntersectionObserverEntry>[]) => void;

let observerCallback: IntersectionCallback;
let observerDisconnect: jest.Mock;
let observerObserve: jest.Mock;
const MockIntersectionObserver = jest.fn();

beforeEach(() => {
  jest.clearAllMocks();
  jest.spyOn(Date, 'now').mockReturnValue(1_000_000);

  observerDisconnect = jest.fn();
  observerObserve = jest.fn();

  MockIntersectionObserver.mockImplementation((callback: IntersectionCallback) => {
    observerCallback = callback;
    return {
      observe: observerObserve,
      disconnect: observerDisconnect,
      unobserve: jest.fn(),
    };
  });

  window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver;
});

afterEach(() => {
  jest.restoreAllMocks();
});

// ── Test component that renders a real DOM element ─────────────────────────

let lastTrackInteraction: ReturnType<typeof useWidgetTracking>['trackInteraction'];

function TestWidget({ name }: { name: string }) {
  const { ref, trackInteraction } = useWidgetTracking(name);
  lastTrackInteraction = trackInteraction;
  return <div ref={ref} data-testid="widget" />;
}

describe('useWidgetTracking', () => {
  it('creates an IntersectionObserver with 50% threshold', () => {
    render(<TestWidget name="test_widget" />);
    expect(MockIntersectionObserver).toHaveBeenCalledWith(expect.any(Function), {
      threshold: 0.5,
    });
  });

  it('observes the DOM element', () => {
    render(<TestWidget name="test_widget" />);
    expect(observerObserve).toHaveBeenCalledTimes(1);
  });

  it('tracks widget_visible when element becomes visible', () => {
    render(<TestWidget name="market_overview" />);

    act(() => {
      observerCallback([{ isIntersecting: true } as IntersectionObserverEntry]);
    });

    expect(mockTrack).toHaveBeenCalledWith('widget_visible', {
      widget: 'market_overview',
    });
  });

  it('tracks widget_visible only once (ignores subsequent intersections)', () => {
    render(<TestWidget name="market_overview" />);

    act(() => {
      observerCallback([{ isIntersecting: true } as IntersectionObserverEntry]);
    });
    act(() => {
      observerCallback([{ isIntersecting: true } as IntersectionObserverEntry]);
    });

    const visibleCalls = mockTrack.mock.calls.filter(
      ([event]: [string]) => event === 'widget_visible'
    );
    expect(visibleCalls).toHaveLength(1);
  });

  it('does not track widget_visible when not intersecting', () => {
    render(<TestWidget name="test_widget" />);

    act(() => {
      observerCallback([{ isIntersecting: false } as IntersectionObserverEntry]);
    });

    expect(mockTrack).not.toHaveBeenCalledWith('widget_visible', expect.anything());
  });

  it('tracks time_on_widget on unmount after visibility', () => {
    (Date.now as jest.Mock).mockReturnValue(1_000_000);

    const { unmount } = render(<TestWidget name="news_feed" />);

    act(() => {
      observerCallback([{ isIntersecting: true } as IntersectionObserverEntry]);
    });

    // Unmount 5 seconds later
    (Date.now as jest.Mock).mockReturnValue(1_005_000);
    unmount();

    expect(mockTrack).toHaveBeenCalledWith('time_on_widget', {
      widget: 'news_feed',
      seconds: 5,
    });
  });

  it('does not track time_on_widget on unmount if never visible', () => {
    const { unmount } = render(<TestWidget name="test_widget" />);
    unmount();

    expect(mockTrack).not.toHaveBeenCalledWith('time_on_widget', expect.anything());
  });

  it('trackInteraction fires widget_interaction event', () => {
    render(<TestWidget name="sector_heatmap" />);

    act(() => {
      lastTrackInteraction('tab_click', { tab: 'US Indices' });
    });

    expect(mockTrack).toHaveBeenCalledWith('widget_interaction', {
      widget: 'sector_heatmap',
      action: 'tab_click',
      tab: 'US Indices',
    });
  });

  it('disconnects observer on unmount', () => {
    const { unmount } = render(<TestWidget name="test_widget" />);
    unmount();
    expect(observerDisconnect).toHaveBeenCalled();
  });
});
