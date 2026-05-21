'use client'

import { useEffect, useState } from 'react'
import { BarChart3, ChevronRight } from 'lucide-react'
import { useSubmissions } from '../hooks/useSubmissions'
import { EmptyState } from './EmptyState'
import { SubmissionStatusBadge } from './SubmissionStatusBadge'
import Link from 'next/link'

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

  useEffect(() => {
    setMyTeamName(localStorage.getItem('iicpc_team_name'))
  }, [])

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
          <EmptyState
            icon={<BarChart3 className="h-12 w-12" />}
            title="No submissions yet"
            description="Upload your trading engine to get started."
            action={
              <Link href="/upload" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors">
                Submit Code
              </Link>
            }
          />
        )}

        {!loading && submissions.length > 0 && (
          <div className="space-y-4">
            {submissions.map((sub) => {
              const isMyTeam = sub.team_name === myTeamName
              return (
                <Link
                  href={`/submissions/${sub.id}`}
                  key={sub.id}
                  className={`block bg-slate-900/50 rounded-lg border p-6 transition-all hover:bg-slate-800/80 ${
                    isMyTeam
                      ? 'border-blue-500/50 shadow-[0_0_15px_rgba(59,130,246,0.1)] hover:border-blue-500'
                      : 'border-slate-700 hover:border-slate-600'
                  }`}
                >
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex items-center gap-4">
                      <SubmissionStatusBadge status={sub.status} />
                      <span className="text-slate-400 font-mono text-sm">{sub.id}</span>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className="text-slate-500 text-sm">
                        {new Date(sub.created_at).toLocaleString()}
                      </span>
                      <ChevronRight className="w-5 h-5 text-slate-600" />
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <p className="text-slate-500 text-xs uppercase tracking-wider mb-1">Team</p>
                      <p className="text-slate-200 font-medium">{sub.team_name}</p>
                    </div>
                    <div>
                      <p className="text-slate-500 text-xs uppercase tracking-wider mb-1">Language</p>
                      <p className="text-slate-200 font-medium">{sub.language.toUpperCase()}</p>
                    </div>
                    <div>
                      <p className="text-slate-500 text-xs uppercase tracking-wider mb-1">Version</p>
                      <p className="text-slate-200 font-medium">v{sub.version || 1}</p>
                    </div>
                  </div>
                </Link>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
