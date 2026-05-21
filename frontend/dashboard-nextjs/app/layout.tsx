import type { Metadata } from 'next'
import './globals.css'
import { Providers } from './providers'
import { Header } from './components/Header'
import { Navbar } from './components/Navbar'

export const metadata: Metadata = {
  title: 'IICPC Dashboard - Distributed Benchmarking Platform',
  description: 'Real-time leaderboard and submission management for IICPC Summer Hackathon 2026',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen bg-slate-950 font-sans text-slate-50 selection:bg-blue-500/30">
        <Providers>
          <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
            <Header />
            <Navbar />
            <main className="container mx-auto px-4 py-8">
              {children}
            </main>
          </div>
        </Providers>
      </body>
    </html>
  )
}
