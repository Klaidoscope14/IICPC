'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

const NAV_LINKS = [
  { href: '/', label: 'System Status' },
  { href: '/upload', label: 'Submit Code' },
  { href: '/leaderboard', label: 'Leaderboard' },
  { href: '/benchmarks', label: 'My Submissions' },
  { href: '/monitor', label: '⚡ Live Monitor' },
]

/**
 * Horizontal navigation bar utilizing Next.js App Router links.
 */
export function Navbar() {
  const pathname = usePathname()

  return (
    <div className="border-b border-slate-700 bg-slate-900/30">
      <div className="container mx-auto px-4">
        <div className="flex gap-1 overflow-x-auto">
          {NAV_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={`px-6 py-3 text-sm font-medium transition-colors whitespace-nowrap ${
                pathname === link.href
                  ? 'text-blue-400 border-b-2 border-blue-400'
                  : 'text-slate-400 hover:text-slate-300'
              }`}
            >
              {link.label}
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}
