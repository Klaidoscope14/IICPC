'use client'

import { Zap } from 'lucide-react'

/**
 * Application header bar with branding.
 */
export function Header() {
  return (
    <header className="border-b border-slate-700 bg-slate-900/50 backdrop-blur-sm">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Zap className="h-8 w-8 text-blue-400" />
            <h1 className="text-2xl font-bold text-white">IICPC Dashboard</h1>
          </div>
          <div className="flex items-center gap-4">
            <span className="text-slate-400 text-sm">Summer Hackathon 2026</span>
          </div>
        </div>
      </div>
    </header>
  )
}
