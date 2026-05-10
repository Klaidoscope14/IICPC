'use client'

import { useEffect, useState } from 'react'
import { Trophy } from 'lucide-react'
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

  return (
    <div className="max-w-6xl mx-auto">
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700 p-8">
        <div className="flex items-center gap-3 mb-6">
          <Trophy className="h-6 w-6 text-yellow-400" />
          <h2 className="text-xl font-semibold text-white">Live Leaderboard</h2>
        </div>

        {loading && (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400" />
            <span className="ml-3 text-slate-400">Loading leaderboard...</span>
          </div>
        )}

        {error && (
          <div className="px-4 py-3 rounded-lg bg-red-600/20 text-red-400 border border-red-500/30 text-sm">
            {error}
          </div>
        )}

        {!loading && !error && leaderboard.length === 0 && (
          <div className="text-center py-12 text-slate-400">
            <Trophy className="h-12 w-12 mx-auto mb-4 opacity-30" />
            <p className="text-lg font-medium">No benchmark results yet</p>
            <p className="text-sm mt-1">Submit your trading engine to appear on the leaderboard.</p>
          </div>
        )}

        {!loading && leaderboard.length > 0 && (
          <div className="overflow-x-auto">
            <table id="leaderboard-table" className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Rank</th>
                  <th className="text-left py-3 px-4 text-slate-400 font-medium">Team</th>
                  <th className="text-right py-3 px-4 text-slate-400 font-medium">TPS</th>
                  <th className="text-right py-3 px-4 text-slate-400 font-medium">
                    Avg Latency (ms)
                  </th>
                  <th className="text-right py-3 px-4 text-slate-400 font-medium">
                    Correctness (%)
                  </th>
                  <th className="text-right py-3 px-4 text-slate-400 font-medium">Score</th>
                </tr>
              </thead>
              <tbody>
                {leaderboard.map((entry) => {
                  const isMyTeam = entry.team === myTeamName
                  return (
                    <tr
                      key={entry.rank}
                      className={`border-b transition-colors ${
                        isMyTeam
                          ? 'bg-blue-900/40 border-blue-500/50 hover:bg-blue-900/60'
                          : 'border-slate-700/50 hover:bg-slate-700/30'
                      }`}
                    >
                      <td className="py-4 px-4">
                        <span
                          className={`font-bold ${
                            entry.rank <= 3
                              ? 'text-yellow-400'
                              : isMyTeam
                              ? 'text-blue-400'
                              : 'text-white'
                          }`}
                        >
                          #{entry.rank}
                        </span>
                      </td>
                      <td className="py-4 px-4 text-white font-medium">{entry.team}</td>
                      <td className="py-4 px-4 text-right text-white">
                        {entry.tps.toLocaleString()}
                      </td>
                      <td className="py-4 px-4 text-right text-white">
                        {entry.latency.toFixed(2)}
                      </td>
                      <td className="py-4 px-4 text-right text-white">
                        {entry.correctness.toFixed(1)}
                      </td>
                      <td className="py-4 px-4 text-right">
                        <span className="px-3 py-1 bg-blue-600/20 text-blue-400 rounded-full text-sm font-medium">
                          {entry.score.toFixed(1)}
                        </span>
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
