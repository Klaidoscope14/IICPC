import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import './globals.css'
import { Providers } from './providers'
import { Header } from './components/Header'
import { Navbar } from './components/Navbar'

const inter = Inter({ subsets: ['latin'], variable: '--font-inter' })

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
    <html lang="en" suppressHydrationWarning className={`${inter.variable}`}>
      <body className="min-h-screen bg-[#09090b] font-sans text-slate-50 selection:bg-blue-500/30 antialiased overflow-x-hidden">
        <Providers>
          <div className="min-h-screen bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-slate-900 via-[#09090b] to-[#09090b]">
            <div className="sticky top-0 z-50 w-full border-b border-slate-800/60 bg-slate-950/50 backdrop-blur-md">
              <Header />
              <Navbar />
            </div>
            <main className="container mx-auto px-4 py-8 relative z-10">
              {children}
            </main>
          </div>
        </Providers>
      </body>
    </html>
  )
}
