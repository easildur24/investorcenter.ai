import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import Header from '@/components/Header'
import { AuthProvider } from '@/lib/auth/AuthContext'
import { ToastProvider } from '@/lib/hooks/useToast'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'InvestorCenter - Financial Data & Analytics',
  description: 'Professional financial data, charts, and investment analytics platform',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <AuthProvider>
          <ToastProvider>
            <Header />
            <main>{children}</main>
          </ToastProvider>
        </AuthProvider>
      </body>
    </html>
  )
}
