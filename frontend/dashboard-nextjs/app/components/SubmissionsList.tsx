'use client'

import { useEffect, useState } from 'react'
import { BarChart3, CheckCircle, Clock } from 'lucide-react'
import { useSubmissions } from '../hooks/useSubmissions'

/**
 * Displays a list of the user's submissions with status indicators.
 * Highlights submissions belonging to the current user's team.
 */
export function SubmissionsList() {
  const { submissions, loading, error } = useSubmissions()
  const [myTeamName, setMyTeamName] = useState<string | null>(null)

  useEffect(() => {
    setMyTeamName(localStorage.getItem('iicpc_team_name'))
  }, [])

  /** Capitalize the first letter of a status string. */
  const formatStatus = (status: string) =>
    status.charAt(0).toUpperCase() + status.slice(1)

  /** Get the appropriate status badge style. */
  const getStatusStyle = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-600/20 text-green-400'
      case 'running':
      case 'benchmarking':
        return 'bg-blue-600/20 text-blue-400'
      default:
        return 'bg-yellow-600/20 text-yellow-400'
    }
  }

  return (
    <div className="max-w-6xl mx-auto">
      <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700 p-8">
        <div className="flex items-center gap-3 mb-6">
          <BarChart3 className="h-6 w-6 text-green-400" />
          <h2 className="text-xl font-semibold text-white">My Submissions</h2>
        </div>

        {loading && (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-400" />
            <span className="ml-3 text-slate-400">Loading submissions...</span>
          </div>
        )}

        {error && (
          <div className="px-4 py-3 rounded-lg bg-red-600/20 text-red-400 border border-red-500/30 text-sm">
            {error}
          </div>
        )}

        {!loading && !error && submissions.length === 0 && (
          <div className="text-center py-12 text-slate-400">
            <BarChart3 className="h-12 w-12 mx-auto mb-4 opacity-30" />
            <p className="text-lg font-medium">No submissions yet</p>
            <p className="text-sm mt-1">Upload your trading engine to get started.</p>
          </div>
        )}

        {!loading && submissions.length > 0 && (
          <div className="space-y-4">
            {submissions.map((sub) => {
              const isMyTeam = sub.team_name === myTeamName
              return (
                <div
                  key={sub.id}
                  className={`bg-slate-900/50 rounded-lg border p-6 transition-all ${
                    isMyTeam
                      ? 'border-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.15)]'
                      : 'border-slate-700'
                  }`}
                >
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-3">
                      <div
                        className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium ${getStatusStyle(sub.status)}`}
                      >
                        {sub.status === 'completed' ? (
                          <CheckCircle className="h-4 w-4" />
                        ) : (
                          <Clock className="h-4 w-4" />
                        )}
                        {formatStatus(sub.status)}
                      </div>
                      <span className="text-slate-400 text-sm">{sub.id}</span>
                    </div>
                    <span className="text-slate-500 text-sm">
                      {new Date(sub.created_at).toLocaleString()}
                    </span>
                  </div>

                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <p className="text-slate-400 text-sm mb-1">Team</p>
                      <p className="text-white font-medium">{sub.team_name}</p>
                    </div>
                    <div>
                      <p className="text-slate-400 text-sm mb-1">Language</p>
                      <p className="text-white font-medium">{sub.language.toUpperCase()}</p>
                    </div>
                    <div>
                      <p className="text-slate-400 text-sm mb-1">Status</p>
                      <p className="text-white font-medium">{formatStatus(sub.status)}</p>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
