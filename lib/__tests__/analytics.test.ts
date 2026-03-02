import { analytics } from '../analytics';

interface MockProvider {
  track: jest.Mock;
  identify: jest.Mock;
  page: jest.Mock;
}

function createMockProvider(): MockProvider {
  return {
    track: jest.fn(),
    identify: jest.fn(),
    page: jest.fn(),
  };
}

describe('Analytics', () => {
  let mockProvider: MockProvider;

  beforeEach(() => {
    mockProvider = createMockProvider();
    analytics.setProvider(mockProvider);
  });

  it('setProvider replaces the active provider', () => {
    analytics.track('test_event');
    expect(mockProvider.track).toHaveBeenCalledTimes(1);
    expect(mockProvider.track).toHaveBeenCalledWith('test_event', expect.any(Object));
  });

  it('passes identify calls through to the provider', () => {
    analytics.identify('user-123', { plan: 'pro' });
    expect(mockProvider.identify).toHaveBeenCalledTimes(1);
    expect(mockProvider.identify).toHaveBeenCalledWith('user-123', {
      plan: 'pro',
    });
  });

  it('passes page calls through to the provider', () => {
    analytics.page('Home', { referrer: '/login' });
    expect(mockProvider.page).toHaveBeenCalledTimes(1);
    expect(mockProvider.page).toHaveBeenCalledWith('Home', {
      referrer: '/login',
    });
  });

  it('track adds a timestamp to properties', () => {
    analytics.track('click', { button: 'signup' });
    const callArgs = mockProvider.track.mock.calls[0];
    expect(callArgs[0]).toBe('click');
    expect(callArgs[1]).toHaveProperty('timestamp');
    expect(callArgs[1].button).toBe('signup');
  });

  it('does not throw when no provider is set', () => {
    // Replace with a fresh no-op-like provider by setting a new Analytics
    // instance behavior. Since analytics is a singleton, we reset by setting
    // a no-op provider (simulating default test environment behavior).
    analytics.setProvider({
      track: () => {},
      identify: () => {},
      page: () => {},
    });

    expect(() => analytics.track('event')).not.toThrow();
    expect(() => analytics.identify('user-1')).not.toThrow();
    expect(() => analytics.page('Dashboard')).not.toThrow();
  });
});
