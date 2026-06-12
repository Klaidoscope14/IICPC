'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

const NAV_LINKS = [
  { href: '/dashboard', label: 'System Status' },
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
    <div className="bg-transparent pb-2">
      <div className="container mx-auto px-4">
        <div className="flex gap-2 overflow-x-auto hide-scrollbar">
          {NAV_LINKS.map((link) => {
            const isActive = pathname === link.href
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`relative px-5 py-2.5 text-sm font-medium transition-all duration-300 whitespace-nowrap rounded-lg overflow-hidden group ${
                  isActive
                    ? 'text-white'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
                }`}
              >
                {isActive && (
                  <div className="absolute inset-0 bg-gradient-to-r from-blue-600/20 to-indigo-600/20 border border-blue-500/30 rounded-lg -z-10"></div>
                )}
                {isActive && (
                  <div className="absolute bottom-0 left-1/4 right-1/4 h-[2px] bg-blue-400 shadow-[0_0_8px_rgba(96,165,250,0.8)] rounded-t-md"></div>
                )}
                <span className="relative z-10">{link.label}</span>
              </Link>
            )
          })}
        </div>
      </div>
    </div>
  )
}
