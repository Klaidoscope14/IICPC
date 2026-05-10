'use client'

import { useState, useEffect } from 'react'
import { apiClient } from '../lib/api'
import type { Submission } from '../types'

const POLL_INTERVAL_MS = 30_000

interface UseSubmissionsResult {
  submissions: Submission[]
  loading: boolean
  error: string | null
}

/**
 * Hook that fetches and polls submissions from the API.
 * Returns submissions, loading state, and any error message.
 */
export function useSubmissions(): UseSubmissionsResult {
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchSubmissions = async () => {
      try {
        const responseData = await apiClient<{ submissions?: Submission[] } | Submission[]>(
          '/api/v1/submissions'
        )
        const submissionsArray = Array.isArray(responseData)
          ? responseData
          : (responseData.submissions || [])
        setSubmissions(submissionsArray)
        setError(null)
      } catch (err) {
        console.error('Failed to fetch submissions:', err)
        setError('Failed to connect to submission service.')
      } finally {
        setLoading(false)
      }
    }

    fetchSubmissions()
    const interval = setInterval(fetchSubmissions, POLL_INTERVAL_MS)
    return () => clearInterval(interval)
  }, [])

  return { submissions, loading, error }
}
