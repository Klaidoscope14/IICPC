'use client'

import useSWR from 'swr'
import { Terminal } from 'lucide-react'

interface LogEntry {
  timestamp?: string
  created_at?: string
  level: string
  message: string
  log_type?: string
}

function formatTimestamp(entry: LogEntry): string {
  const raw = entry.created_at || entry.timestamp
  if (!raw) return '--:--:--'
  try {
    const d = new Date(raw)
    if (isNaN(d.getTime())) return raw
    return d.toISOString().split('T')[1].replace('Z', '')
  } catch {
    return raw
  }
}

export function LogsViewer({ submissionId }: { submissionId: string }) {
  const { data, error, isLoading } = useSWR<{ logs: LogEntry[] }>(
    `/api/v1/submissions/${submissionId}/logs`
  )

  const logs = data?.logs || []

  return (
    <div className="bg-slate-900 rounded-xl border border-slate-700 overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-3 bg-slate-800 border-b border-slate-700">
        <Terminal className="w-5 h-5 text-slate-400" />
        <h3 className="font-mono text-sm font-semibold text-slate-200">Build & Container Logs</h3>
      </div>
      
      <div className="p-4 h-96 overflow-y-auto font-mono text-sm">
        {isLoading && (
          <div className="flex items-center gap-2 text-slate-500">
            <span className="animate-pulse">Loading logs...</span>
          </div>
        )}
        
        {error && (
          <div className="text-red-400">Failed to load logs.</div>
        )}

        {!isLoading && !error && logs.length === 0 && (
          <div className="text-slate-500">No logs available yet.</div>
        )}

        {!isLoading && !error && logs.length > 0 && (
          <div className="space-y-1">
            {logs.map((log, i) => (
              <div key={i} className="flex gap-4">
                <span className="text-slate-500 shrink-0">
                  {formatTimestamp(log)}
                </span>
                <span className={`shrink-0 w-12 ${
                  log.level === 'error' ? 'text-red-400' :
                  log.level === 'warn' ? 'text-yellow-400' :
                  'text-blue-400'
                }`}>
                  [{log.level.toUpperCase()}]
                </span>
                <span className="text-slate-300 break-all">{log.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

