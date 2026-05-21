'use client'

import useSWR from 'swr'
import type { Submission } from '../types'

const POLL_INTERVAL_MS = 10_000

interface UseSubmissionsResult {
  submissions: Submission[]
  loading: boolean
  error: string | null
}

export function useSubmissions(): UseSubmissionsResult {
  const { data, error, isLoading } = useSWR<{ submissions?: Submission[] } | Submission[]>(
    '/api/v1/submissions',
    { refreshInterval: POLL_INTERVAL_MS }
  )

  const submissions = data 
    ? (Array.isArray(data) ? data : (data.submissions || []))
    : []

  return { 
    submissions, 
    loading: isLoading, 
    error: error ? (error.message || 'Failed to connect to submission service.') : null 
  }
}
