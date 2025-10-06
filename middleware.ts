import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  // Add no-cache headers for ticker pages to prevent browser caching of stale prices
  if (request.nextUrl.pathname.startsWith('/ticker/')) {
    const response = NextResponse.next();

    // Comprehensive cache-busting headers
    response.headers.set('Cache-Control', 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0, s-maxage=0');
    response.headers.set('Pragma', 'no-cache');
    response.headers.set('Expires', '0');
    response.headers.set('Surrogate-Control', 'no-store');

    return response;
  }

  return NextResponse.next();
}

export const config = {
  matcher: '/ticker/:path*',
};
