'use client'

import { useState, useEffect, useRef, useCallback } from 'react'

const WS_BASE_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081'
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
 * Connects to ws://localhost:8081/ws/benchmarks/:id/stream
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

  const connect = useCallback(() => {
    if (!benchmarkId) return

    shouldConnect.current = true
    setError(null)
    setConnected(false)
    setMetrics(null)
    setMetricsHistory([])

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
            return
          }

          // Check for error.
          if (data.error) {
            setError(data.error)
            return
          }

          setMetrics(data)
          setMetricsHistory(prev => [...prev.slice(-120), data]) // Keep last 2 minutes.
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
