'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { Activity, Trophy, Crown } from 'lucide-react'
import { useLeaderboard } from '../hooks/useLeaderboard'

/**
 * Live leaderboard table showing ranked submissions with benchmark metrics.
 * Highlights the current user's team row.
 */
export function Leaderboard() {
  const { leaderboard, loading, error } = useLeaderboard()
  const [myTeamName, setMyTeamName] = useState<string | null>(null)

  useEffect(() => {
    setMyTeamName(localStorage.getItem('iicpc_team_name'))
  }, [])

  const getRankStyle = (rank: number) => {
    switch (rank) {
      case 1: return { color: 'text-yellow-400 text-glow-gold', icon: <Crown className="w-5 h-5 text-yellow-400 drop-shadow-[0_0_8px_rgba(250,204,21,0.8)] inline-block mr-2" /> }
      case 2: return { color: 'text-slate-300 text-glow-silver', icon: null }
      case 3: return { color: 'text-amber-600 text-glow-bronze', icon: null }
      default: return { color: 'text-slate-400', icon: null }
    }
  }

  return (
    <div className="max-w-6xl mx-auto animate-in fade-in slide-in-from-bottom-4 duration-700">
      <div className="glass-panel rounded-2xl p-8 relative overflow-hidden">
        {/* Subtle background glow */}
        <div className="absolute -top-40 -right-40 w-96 h-96 bg-blue-600/10 rounded-full blur-[100px] pointer-events-none"></div>

        <div className="flex items-center gap-4 mb-8">
          <div className="p-3 bg-yellow-500/10 rounded-xl border border-yellow-500/20">
            <Trophy className="h-8 w-8 text-yellow-400 drop-shadow-[0_0_8px_rgba(250,204,21,0.8)]" />
          </div>
          <h2 className="text-3xl font-bold text-white tracking-tight">Live Leaderboard</h2>
        </div>

        {loading && (
          <div className="flex flex-col items-center justify-center py-20">
            <div className="relative w-12 h-12">
              <div className="absolute inset-0 rounded-full border-t-2 border-blue-400 animate-spin"></div>
              <div className="absolute inset-2 rounded-full border-r-2 border-indigo-400 animate-spin opacity-50 animation-delay-150"></div>
            </div>
            <span className="mt-6 text-slate-400 font-medium tracking-widest text-sm uppercase">Synchronizing...</span>
          </div>
        )}

        {error && (
          <div className="px-5 py-4 rounded-xl bg-red-950/40 text-red-400 border border-red-500/30 text-sm backdrop-blur-md">
            {error}
          </div>
        )}

        {!loading && !error && leaderboard.length === 0 && (
          <div className="text-center py-24 text-slate-400 bg-slate-900/30 rounded-xl border border-dashed border-slate-700">
            <Trophy className="h-16 w-16 mx-auto mb-6 opacity-20" />
            <p className="text-xl font-medium text-slate-300">No benchmark results yet</p>
            <p className="text-sm mt-2 text-slate-500">Submit your trading engine to appear on the leaderboard.</p>
          </div>
        )}

        {!loading && leaderboard.length > 0 && (
          <div className="overflow-x-auto rounded-xl border border-slate-700/50 bg-slate-900/50">
            <table id="leaderboard-table" className="w-full">
              <thead>
                <tr className="border-b border-slate-700/80 bg-slate-950/80">
                  <th className="text-left py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">Rank</th>
                  <th className="text-left py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">Team</th>
                  <th className="text-right py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">TPS</th>
                  <th className="text-right py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">Avg Latency (ms)</th>
                  <th className="text-right py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">Correctness (%)</th>
                  <th className="text-right py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">Score</th>
                  <th className="text-right py-4 px-6 text-slate-400 font-semibold text-sm uppercase tracking-wider">Live</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/50">
                {leaderboard.map((entry) => {
                  const isMyTeam = entry.team === myTeamName
                  const rankStyle = getRankStyle(entry.rank)
                  
                  return (
                    <tr
                      key={entry.benchmarkId || `${entry.team}-${entry.rank}`}
                      className={`group transition-all duration-300 ${
                        isMyTeam
                          ? 'bg-blue-900/20 hover:bg-blue-900/40 relative outline outline-1 outline-blue-500/50 shadow-[inset_0_0_20px_rgba(59,130,246,0.1)]'
                          : 'hover:bg-slate-800/60'
                      }`}
                    >
                      <td className="py-5 px-6 whitespace-nowrap">
                        <div className="flex items-center font-bold text-lg">
                          {rankStyle.icon}
                          <span className={rankStyle.color}>#{entry.rank}</span>
                        </div>
                      </td>
                      <td className="py-5 px-6">
                        <span className={`font-semibold ${isMyTeam ? 'text-blue-300' : 'text-slate-200'}`}>
                          {entry.team}
                        </span>
                        {isMyTeam && <span className="ml-3 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-500/20 text-blue-400 border border-blue-500/30">YOU</span>}
                      </td>
                      <td className="py-5 px-6 text-right font-mono text-slate-300">
                        {entry.tps.toLocaleString()}
                      </td>
                      <td className="py-5 px-6 text-right font-mono text-slate-300">
                        {entry.latency.toFixed(2)}
                      </td>
                      <td className="py-5 px-6 text-right font-mono text-slate-300">
                        <span className={entry.correctness >= 99 ? 'text-emerald-400' : entry.correctness >= 90 ? 'text-amber-400' : 'text-rose-400'}>
                          {entry.correctness.toFixed(1)}%
                        </span>
                      </td>
                      <td className="py-5 px-6 text-right">
                        <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-bold ${
                          entry.rank === 1 ? 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30' :
                          'bg-indigo-500/10 text-indigo-300 border border-indigo-500/20'
                        }`}>
                          {entry.score.toFixed(1)}
                        </span>
                      </td>
                      <td className="py-5 px-6 text-right relative z-20">
                        {entry.benchmarkId ? (
                          <Link
                            href={`/monitor?benchmarkId=${entry.benchmarkId}`}
                            title="Open live monitor"
                            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-600 bg-slate-800 text-slate-400 transition-all hover:bg-blue-600 hover:text-white hover:border-blue-500 hover:shadow-[0_0_15px_rgba(37,99,235,0.5)]"
                          >
                            <Activity className="h-4 w-4" />
                          </Link>
                        ) : (
                          <span className="text-sm text-slate-600">--</span>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
