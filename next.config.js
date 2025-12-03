/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  experimental: {
    outputFileTracingRoot: undefined,
  },
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'api.polygon.io',
        pathname: '/v1/reference/company-branding/**',
      },
    ],
  },
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || '/api/v1',
  },
  transpilePackages: ['react-gauge-chart'],
  webpack: (config) => {
    // Ensure lodash is properly resolved
    config.resolve.alias = {
      ...config.resolve.alias,
      'lodash/isEqual': require.resolve('lodash/isEqual'),
    };
    return config;
  },
  async rewrites() {
    // Only proxy in development - production uses ingress routing
    if (process.env.NODE_ENV === 'development') {
      return [
        {
          source: '/api/v1/:path*',
          destination: 'http://localhost:8080/api/v1/:path*',
        },
      ]
    }
    return []
  },
}

module.exports = nextConfig