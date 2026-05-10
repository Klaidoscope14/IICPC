'use client'

import { useState } from 'react'
import { Activity, Radio, AlertCircle, TrendingUp, Clock, Zap, BarChart3 } from 'lucide-react'
import { useBenchmarkStream } from '../hooks/useBenchmarkStream'

/**
 * Live benchmark monitoring dashboard with real-time metrics.
 *
 * Features:
 * - Connect to a benchmark via WebSocket for live updates
 * - Real-time TPS, latency, and error rate display
 * - Sparkline history for key metrics
 * - Auto-disconnects when the benchmark completes
 */
export function BenchmarkMonitor() {
  const [benchmarkId, setBenchmarkId] = useState('')
  const [inputId, setInputId] = useState('')
  const { metrics, metricsHistory, connected, error, connect, disconnect } = useBenchmarkStream(benchmarkId || null)

  const handleConnect = () => {
    if (inputId.trim()) {
      setBenchmarkId(inputId.trim())
      setTimeout(connect, 100)
    }
  }

  const handleDisconnect = () => {
    disconnect()
    setBenchmarkId('')
  }

  return (
    <div className="space-y-6">
      {/* Connection Bar */}
      <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
        <div className="flex items-center gap-3">
          <Radio className={`w-5 h-5 ${connected ? 'text-green-400 animate-pulse' : 'text-slate-500'}`} />
          <input
            id="benchmark-id-input"
            type="text"
            placeholder="Enter Benchmark ID to monitor..."
            value={inputId}
            onChange={e => setInputId(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleConnect()}
            className="flex-1 bg-slate-900/50 border border-slate-600 rounded-lg px-4 py-2 text-slate-200 placeholder:text-slate-500 focus:outline-none focus:border-blue-500 transition-colors"
            disabled={connected}
          />
          {connected ? (
            <button
              id="disconnect-btn"
              onClick={handleDisconnect}
              className="px-4 py-2 bg-red-600/20 text-red-400 border border-red-500/30 rounded-lg hover:bg-red-600/30 transition-colors"
            >
              Disconnect
            </button>
          ) : (
            <button
              id="connect-btn"
              onClick={handleConnect}
              className="px-4 py-2 bg-blue-600/20 text-blue-400 border border-blue-500/30 rounded-lg hover:bg-blue-600/30 transition-colors"
            >
              Connect
            </button>
          )}
        </div>

        {connected && (
          <div className="mt-2 flex items-center gap-2 text-sm text-green-400">
            <span className="w-2 h-2 bg-green-400 rounded-full animate-pulse" />
            Connected — streaming live metrics
          </div>
        )}
        {error && (
          <div className="mt-2 flex items-center gap-2 text-sm text-red-400">
            <AlertCircle className="w-4 h-4" />
            {error}
          </div>
        )}
      </div>

      {/* Metrics Grid */}
      {metrics && (
        <>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <MetricCard
              icon={<Zap className="w-5 h-5 text-yellow-400" />}
              label="Throughput (TPS)"
              value={metrics.metrics.current_tps.toFixed(1)}
              unit="ops/s"
              color="yellow"
              sparklineData={metricsHistory.map(m => m.metrics.current_tps)}
            />
            <MetricCard
              icon={<Clock className="w-5 h-5 text-blue-400" />}
              label="P99 Latency"
              value={metrics.metrics.p99_latency_ms.toFixed(2)}
              unit="ms"
              color="blue"
              sparklineData={metricsHistory.map(m => m.metrics.p99_latency_ms)}
            />
            <MetricCard
              icon={<TrendingUp className="w-5 h-5 text-green-400" />}
              label="Success Rate"
              value={metrics.metrics.total_orders_sent > 0
                ? ((metrics.metrics.total_orders_acknowledged / metrics.metrics.total_orders_sent) * 100).toFixed(1)
                : '0'}
              unit="%"
              color="green"
            />
            <MetricCard
              icon={<AlertCircle className="w-5 h-5 text-red-400" />}
              label="Errors"
              value={metrics.metrics.total_errors.toString()}
              color="red"
            />
          </div>

          {/* Detailed Metrics */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
              <h3 className="text-lg font-semibold text-slate-200 mb-4 flex items-center gap-2">
                <BarChart3 className="w-5 h-5 text-purple-400" />
                Latency Distribution
              </h3>
              <div className="space-y-3">
                <LatencyBar label="P50" value={metrics.metrics.p50_latency_ms} max={Math.max(metrics.metrics.p99_latency_ms * 1.2, 1)} />
                <LatencyBar label="P90" value={metrics.metrics.p90_latency_ms} max={Math.max(metrics.metrics.p99_latency_ms * 1.2, 1)} />
                <LatencyBar label="P99" value={metrics.metrics.p99_latency_ms} max={Math.max(metrics.metrics.p99_latency_ms * 1.2, 1)} />
                <LatencyBar label="Avg" value={metrics.metrics.avg_latency_ms} max={Math.max(metrics.metrics.p99_latency_ms * 1.2, 1)} />
              </div>
            </div>

            <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-6">
              <h3 className="text-lg font-semibold text-slate-200 mb-4 flex items-center gap-2">
                <Activity className="w-5 h-5 text-cyan-400" />
                Benchmark Status
              </h3>
              <div className="space-y-3">
                <StatusRow label="Status" value={metrics.status} />
                <StatusRow label="Elapsed" value={`${metrics.elapsed_time}s`} />
                <StatusRow label="Orders Sent" value={metrics.metrics.total_orders_sent.toLocaleString()} />
                <StatusRow label="Orders Acked" value={metrics.metrics.total_orders_acknowledged.toLocaleString()} />
                <StatusRow label="Bot Count" value={metrics.config.bot_count.toString()} />
                <StatusRow label="Target OPS" value={metrics.config.orders_per_second.toLocaleString()} />
              </div>
            </div>
          </div>
        </>
      )}

      {/* Empty State */}
      {!connected && !metrics && (
        <div className="text-center py-16">
          <Radio className="w-12 h-12 text-slate-600 mx-auto mb-4" />
          <p className="text-slate-400 text-lg">Enter a Benchmark ID to start monitoring</p>
          <p className="text-slate-500 text-sm mt-2">Real-time metrics will stream via WebSocket</p>
        </div>
      )}
    </div>
  )
}

// --- Sub-components ---

interface MetricCardProps {
  icon: React.ReactNode
  label: string
  value: string
  unit?: string
  color: string
  sparklineData?: number[]
}

function MetricCard({ icon, label, value, unit, color, sparklineData }: MetricCardProps) {
  return (
    <div className="bg-slate-800/50 border border-slate-700 rounded-xl p-4">
      <div className="flex items-center gap-2 mb-2 text-sm text-slate-400">
        {icon}
        {label}
      </div>
      <div className="flex items-baseline gap-1">
        <span className="text-2xl font-bold text-slate-100">{value}</span>
        {unit && <span className="text-sm text-slate-400">{unit}</span>}
      </div>
      {sparklineData && sparklineData.length > 1 && (
        <Sparkline data={sparklineData} color={color} />
      )}
    </div>
  )
}

function Sparkline({ data, color }: { data: number[]; color: string }) {
  const width = 200
  const height = 30
  const max = Math.max(...data, 1)
  const min = Math.min(...data, 0)
  const range = max - min || 1

  const points = data.map((v, i) => ({
    x: (i / (data.length - 1)) * width,
    y: height - ((v - min) / range) * height,
  }))

  const pathData = points.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ')

  const colorMap: Record<string, string> = {
    yellow: '#facc15',
    blue: '#60a5fa',
    green: '#4ade80',
    red: '#f87171',
  }

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-8 mt-2">
      <path d={pathData} fill="none" stroke={colorMap[color] || '#94a3b8'} strokeWidth="2" />
    </svg>
  )
}

function LatencyBar({ label, value, max }: { label: string; value: number; max: number }) {
  const width = Math.min((value / max) * 100, 100)

  return (
    <div>
      <div className="flex justify-between text-sm mb-1">
        <span className="text-slate-400">{label}</span>
        <span className="text-slate-200 font-mono">{value.toFixed(2)} ms</span>
      </div>
      <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
        <div
          className="h-full bg-gradient-to-r from-blue-500 to-purple-500 rounded-full transition-all duration-500"
          style={{ width: `${width}%` }}
        />
      </div>
    </div>
  )
}

function StatusRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-sm text-slate-400">{label}</span>
      <span className="text-sm text-slate-200 font-mono">{value}</span>
    </div>
  )
}
