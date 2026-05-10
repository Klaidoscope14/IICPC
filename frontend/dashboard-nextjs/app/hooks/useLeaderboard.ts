'use client'

import { useState, useEffect } from 'react'
import { apiClient } from '../lib/api'
import type { LeaderboardEntry, Submission } from '../types'

const POLL_INTERVAL_MS = 30_000

interface UseLeaderboardResult {
  leaderboard: LeaderboardEntry[]
  loading: boolean
  error: string | null
}

/**
 * Hook that fetches submissions, computes leaderboard rankings, and polls.
 */
export function useLeaderboard(): UseLeaderboardResult {
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchLeaderboard = async () => {
      try {
        const responseData = await apiClient<{ submissions?: Submission[] } | Submission[]>(
          '/api/v1/submissions'
        )
        const submissionsArray = Array.isArray(responseData)
          ? responseData
          : (responseData.submissions || [])

        // Process submissions with benchmark results into leaderboard entries.
        const entries = submissionsArray
          .filter((sub) => sub.benchmark_results && sub.benchmark_results.length > 0)
          .map((sub) => {
            const latestResult = sub.benchmark_results![sub.benchmark_results!.length - 1]
            return {
              rank: 0,
              team: sub.team_name,
              tps: latestResult.metrics?.total_orders_sent || 0,
              latency: latestResult.metrics?.avg_latency_ms || 0,
              correctness: latestResult.metrics?.correctness || 0,
              score: latestResult.score || 0,
            }
          })
          .sort((a, b) => b.score - a.score)
          .map((entry, index) => ({ ...entry, rank: index + 1 }))

        setLeaderboard(entries)
        setError(null)
      } catch (err) {
        console.error('Failed to fetch leaderboard:', err)
        setError('Failed to connect to server.')
      } finally {
        setLoading(false)
      }
    }

    fetchLeaderboard()
    const interval = setInterval(fetchLeaderboard, POLL_INTERVAL_MS)
    return () => clearInterval(interval)
  }, [])

  return { leaderboard, loading, error }
}
