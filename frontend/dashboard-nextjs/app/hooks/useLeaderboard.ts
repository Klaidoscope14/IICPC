'use client'

import useSWR from 'swr'
import type { BackendLeaderboardEntry, LeaderboardEntry } from '../types'

const POLL_INTERVAL_MS = 10_000 // Poll every 10 seconds for real-time feel

interface UseLeaderboardResult {
  leaderboard: LeaderboardEntry[]
  loading: boolean
  error: string | null
}

export function useLeaderboard(): UseLeaderboardResult {
  const { data, error, isLoading } = useSWR<{
    leaderboard?: BackendLeaderboardEntry[] | null
  }>(
    '/api/v1/leaderboard',
    { refreshInterval: POLL_INTERVAL_MS }
  )

  const leaderboard: LeaderboardEntry[] = (data?.leaderboard || []).map((entry, index) => ({
    rank: entry.rank || index + 1,
    submissionId: entry.submission_id,
    benchmarkId: entry.benchmark_id,
    team: entry.team_name,
    tps: entry.tps,
    latency: entry.p99_latency_ms,
    correctness: entry.correctness_score,
    score: entry.composite_score,
  }))

  return { 
    leaderboard, 
    loading: isLoading, 
    error: error ? (error.message || 'Failed to connect to server.') : null 
  }
}
