'use client'

import { useEffect, useState } from 'react'
import { Activity, Server, Database, Network, Shield, Cpu, AlertCircle } from 'lucide-react'

interface HealthResponse {
  status: string
  service: string
  version: string
  checks: Record<string, string>
  timestamp: string
}

type ServiceStatus = 'operational' | 'degraded' | 'down' | 'loading'

/**
 * Home Page - System Status Dashboard
 * Fetches real health data from the API gateway.
 */
export default function Home() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchHealth = async () => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082'
      const res = await fetch(`${apiUrl}/health`, { cache: 'no-store' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: HealthResponse = await res.json()
      setHealth(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reach API gateway')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchHealth()
    const interval = setInterval(fetchHealth, 10000)
    return () => clearInterval(interval)
  }, [])

  const getCheckStatus = (checkName: string): ServiceStatus => {
    if (loading) return 'loading'
    if (!health) return 'down'
    const check = health.checks[checkName]
    return check === 'ok' ? 'operational' : 'down'
  }

  const gatewayStatus: ServiceStatus = loading ? 'loading' : health?.status === 'healthy' ? 'operational' : 'down'

  return (
    <div className="max-w-6xl mx-auto space-y-8">
      <div className="flex items-center gap-3 mb-8">
        <Activity className="h-8 w-8 text-blue-400" />
        <div>
          <h2 className="text-3xl font-bold text-white">System Status</h2>
          <p className="text-slate-400 mt-1">
            Real-time health of the IICPC platform infrastructure
            {health && (
              <span className="text-slate-500 text-xs ml-2">
                Last checked: {new Date(health.timestamp).toLocaleTimeString()}
              </span>
            )}
          </p>
        </div>
      </div>

      {error && (
        <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-red-600/20 text-red-400 border border-red-500/30 text-sm">
          <AlertCircle className="w-5 h-5 shrink-0" />
          <span>Failed to reach API gateway: {error}</span>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <StatusCard 
          title="API Gateway" 
          icon={<Network className="w-5 h-5 text-indigo-400" />}
          status={gatewayStatus}
          metrics={[
            { label: 'Version', value: health?.version || '--' },
            { label: 'Status', value: health?.status || '--' },
          ]}
        />
        <StatusCard 
          title="Submission Service" 
          icon={<Server className="w-5 h-5 text-cyan-400" />}
          status={getCheckStatus('submission-service')}
          metrics={[
            { label: 'Health', value: health?.checks['submission-service'] || '--' },
          ]}
        />
        <StatusCard 
          title="Validation Service" 
          icon={<Shield className="w-5 h-5 text-green-400" />}
          status={getCheckStatus('validation-service')}
          metrics={[
            { label: 'Health', value: health?.checks['validation-service'] || '--' },
          ]}
        />
        <StatusCard 
          title="Benchmark Orchestrator" 
          icon={<Cpu className="w-5 h-5 text-purple-400" />}
          status={getCheckStatus('benchmark-orchestrator')}
          metrics={[
            { label: 'Health', value: health?.checks['benchmark-orchestrator'] || '--' },
          ]}
        />
        <StatusCard 
          title="Bot Fleet" 
          icon={<Activity className="w-5 h-5 text-yellow-400" />}
          status={getCheckStatus('bot-fleet')}
          metrics={[
            { label: 'Health', value: health?.checks['bot-fleet'] || '--' },
          ]}
        />
        <StatusCard 
          title="Redpanda Message Bus" 
          icon={<Network className="w-5 h-5 text-red-400" />}
          status={gatewayStatus}
          metrics={[
            { label: 'Broker', value: gatewayStatus === 'operational' ? 'Connected' : '--' },
          ]}
        />
      </div>
    </div>
  )
}

function StatusCard({ title, icon, status, metrics }: { title: string, icon: React.ReactNode, status: ServiceStatus, metrics: {label: string, value: string}[] }) {
  const statusColors = {
    operational: 'bg-green-500/20 text-green-400 border-green-500/30',
    degraded: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
    down: 'bg-red-500/20 text-red-400 border-red-500/30',
    loading: 'bg-slate-500/20 text-slate-400 border-slate-500/30',
  }

  const statusLabel = {
    operational: 'OPERATIONAL',
    degraded: 'DEGRADED',
    down: 'DOWN',
    loading: 'CHECKING...',
  }

  return (
    <div className="bg-slate-800/50 backdrop-blur-sm rounded-xl border border-slate-700 p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2 text-slate-200 font-semibold">
          {icon}
          {title}
        </div>
        <div className={`px-2 py-1 rounded-full text-xs font-medium border uppercase tracking-wider ${statusColors[status]}`}>
          {statusLabel[status]}
        </div>
      </div>
      <div className="space-y-3 mt-6">
        {metrics.map((m, i) => (
          <div key={i} className="flex justify-between items-center text-sm">
            <span className="text-slate-400">{m.label}</span>
            <span className="text-slate-200 font-mono">{m.value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
