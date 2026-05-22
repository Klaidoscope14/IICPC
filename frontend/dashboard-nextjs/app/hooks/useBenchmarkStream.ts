'use client'

import { useState, useEffect, useRef, useCallback } from 'react'

import { apiClient } from '../lib/api'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

function toWebSocketBaseUrl(httpBaseUrl: string): string {
  try {
    const url = new URL(httpBaseUrl)
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
    return url.origin
  } catch {
    return 'ws://localhost:8082'
  }
}

const WS_BASE_URL = process.env.NEXT_PUBLIC_WS_URL || toWebSocketBaseUrl(API_BASE_URL)
const RECONNECT_DELAY_MS = 3000
const MAX_RECONNECT_ATTEMPTS = 5

export interface BenchmarkMetrics {
  id: string
  submission_id: string
  deployment_id: string
  status: string
  config: {
    bot_count: number
    duration_seconds: number
    orders_per_second: number
    protocols: string[]
  }
  started_at: string
  completed_at?: string
  elapsed_time: number
  metrics: {
    current_tps: number
    avg_latency_ms: number
    total_orders_sent: number
    total_orders_acknowledged: number
    total_errors: number
    p50_latency_ms: number
    p90_latency_ms: number
    p99_latency_ms: number
    active_connections: number
    cpu_usage_percent: number
    memory_usage_mb: number
  }
}

interface UseBenchmarkStreamResult {
  metrics: BenchmarkMetrics | null
  metricsHistory: BenchmarkMetrics[]
  connected: boolean
  error: string | null
  connect: () => void
  disconnect: () => void
}

/**
 * WebSocket hook for streaming real-time benchmark telemetry.
 *
 * Connects to {WS_BASE_URL}/ws/benchmarks/:id/stream
 * and receives JSON snapshots every second.
 */
export function useBenchmarkStream(benchmarkId: string | null): UseBenchmarkStreamResult {
  const [metrics, setMetrics] = useState<BenchmarkMetrics | null>(null)
  const [metricsHistory, setMetricsHistory] = useState<BenchmarkMetrics[]>([])
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttempts = useRef(0)
  const shouldConnect = useRef(false)

  const connect = useCallback(async () => {
    if (!benchmarkId) return

    shouldConnect.current = true
    setError(null)
    setConnected(false)
    setMetrics(null)
    setMetricsHistory([])

    try {
      // Fetch initial benchmark state to populate config, status, etc.
      const initialData = await apiClient<BenchmarkMetrics>(`/api/v1/benchmarks/${benchmarkId}`)
      setMetrics(initialData)
      setMetricsHistory([initialData])
    } catch (err) {
      console.warn('Failed to fetch initial benchmark state:', err)
      setError('No valid submission found with this ID.')
      shouldConnect.current = false
      return
    }

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    const url = `${WS_BASE_URL}/ws/benchmarks/${benchmarkId}/stream`

    try {
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        setConnected(true)
        setError(null)
        reconnectAttempts.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)

          // Check for benchmark ended event.
          if (data.event === 'benchmark_ended') {
            shouldConnect.current = false
            setMetrics(prev => prev ? { ...prev, status: 'completed' } : prev)
            return
          }

          if (data.event === 'benchmark_started') {
            setMetrics(prev => prev ? {
              ...prev,
              status: 'running',
              started_at: data.data.started_at,
              config: data.data.config
            } : null)
            return
          }

          // Check for error.
          if (data.error) {
            setError(data.error)
            return
          }

          // Handle TelemetrySnapshotEvent
          if (data.metrics) {
            setMetrics(prev => {
              if (!prev) return null // Wait for initial state fetch
              const timestamp = data.timestamp ? new Date(data.timestamp).getTime() : Date.now()
              const startedAt = new Date(prev.started_at).getTime()
              const elapsed = Math.max(0, Math.floor((timestamp - startedAt) / 1000))
              
              const updated = {
                ...prev,
                elapsed_time: elapsed > prev.elapsed_time ? elapsed : prev.elapsed_time,
                metrics: { ...prev.metrics, ...data.metrics }
              }
              return updated
            })

            setMetricsHistory(prev => {
              const last = prev[prev.length - 1]
              if (!last) return prev
              const updated = { ...last, metrics: { ...last.metrics, ...data.metrics } }
              return [...prev.slice(-119), updated]
            })
          }
        } catch {
          setError('Failed to parse metrics data')
        }
      }

      ws.onerror = () => {
        setError('WebSocket connection error')
      }

      ws.onclose = () => {
        setConnected(false)
        wsRef.current = null

        // Auto-reconnect if we should still be connected.
        if (shouldConnect.current && reconnectAttempts.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts.current++
          setTimeout(connect, RECONNECT_DELAY_MS)
        }
      }
    } catch {
      setError('Failed to create WebSocket connection')
    }
  }, [benchmarkId])

  const disconnect = useCallback(() => {
    shouldConnect.current = false
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setConnected(false)
    setMetrics(null)
    setMetricsHistory([])
  }, [])

  // Clean up on unmount.
  useEffect(() => {
    return () => {
      shouldConnect.current = false
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [])

  return { metrics, metricsHistory, connected, error, connect, disconnect }
}
