import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from './providers'

const inter = Inter({ subsets: ['latin'], variable: '--font-inter' })

export const metadata: Metadata = {
  title: 'IICPC Distributed Benchmarking Platform',
  description: 'Real-time leaderboard and submission management for IICPC Summer Hackathon 2026',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning className={`${inter.variable}`}>
      <body className="min-h-screen font-sans antialiased overflow-x-hidden text-slate-900 bg-white">
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  )
}
