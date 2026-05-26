'use client'

import { Zap, Timer } from 'lucide-react'

/**
 * Application header bar with branding and contest timer.
 */
export function Header() {
  return (
    <header className="border-b-0 bg-transparent">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3 group cursor-pointer">
            <div className="relative">
              <div className="absolute inset-0 bg-blue-500 blur-md opacity-50 group-hover:opacity-100 transition-opacity duration-500 rounded-full"></div>
              <Zap className="relative h-8 w-8 text-blue-400 drop-shadow-[0_0_8px_rgba(96,165,250,0.8)]" />
            </div>
            <h1 className="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-400 tracking-tight text-glow">
              IICPC Dashboard
            </h1>
          </div>
          <div className="flex items-center gap-6">
            <div className="hidden md:flex items-center gap-2 px-4 py-1.5 rounded-full bg-slate-800/50 border border-slate-700/50 shadow-inner">
              <Timer className="w-4 h-4 text-emerald-400" />
              <span className="text-emerald-400 font-mono text-sm font-medium tracking-wider">04:23:59</span>
              <span className="text-slate-500 text-xs ml-1">REMAINING</span>
            </div>
            <span className="text-slate-400 text-sm font-medium tracking-wide">Summer Hackathon 2026</span>
          </div>
        </div>
      </div>
    </header>
  )
}
